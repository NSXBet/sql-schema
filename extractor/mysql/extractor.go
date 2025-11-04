package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/blang/semver/v4"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

	"github.com/nsxbet/sql-schema"
	"github.com/nsxbet/sql-schema/comparer"
)

const (
	autoIncrementSymbol    = "AUTO_INCREMENT"
	autoRandSymbol         = "AUTO_RANDOM"
	pkAutoRandomBitsSymbol = "PK_AUTO_RANDOM_BITS"
	virtualGenerated       = "VIRTUAL GENERATED"
	storedGenerated        = "STORED GENERATED"
	baseTableType          = "BASE TABLE"
	viewTableType          = "VIEW"
)

var (
	systemDatabases = map[string]bool{
		"information_schema": true,
		"mysql":              true,
		"performance_schema": true,
		"sys":                true,
		// OceanBase only
		"oceanbase":  true,
		"SYS":        true,
		"LBACSYS":    true,
		"ORAAUDITOR": true,
		"__public":   true,
	}

	viewDefMatcher = regexp.MustCompile("CREATE ALGORITHM=(UNDEFINED|MERGE|TEMPTABLE) DEFINER=`([^`]+)`@`([^`]+)` SQL SECURITY (DEFINER|INVOKER) VIEW `([^`]+)`( \\((`([^`]+)`)+\\))? AS (?P<def>.+)")
)

// Extractor extracts MySQL database schema.
type Extractor struct {
	db           *sql.DB
	databaseName string
}

// NewExtractor creates a new MySQL schema extractor.
func NewExtractor(db *sql.DB, databaseName string) *Extractor {
	return &Extractor{
		db:           db,
		databaseName: databaseName,
	}
}

// ExtractSchema extracts the complete schema metadata for the database.
func (e *Extractor) ExtractSchema(ctx context.Context) (*schemaextract.DatabaseSchema, error) {
	// Query MySQL version
	version, rest, err := e.getVersion(ctx)
	if err != nil {
		return nil, err
	}
	semVersion, err := semver.Make(version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MySQL version %s to semantic version: %w", version, err)
	}
	atLeast8_0_13 := semVersion.GE(semver.MustParse("8.0.13"))
	atLeast8_0_16 := semVersion.GE(semver.MustParse("8.0.16"))
	atLeast5_7_0 := semVersion.GE(semver.MustParse("5.7.0"))

	schemaMetadata := &schemaextract.Schema{
		Name: "",
	}

	// Query index info
	indexMap, err := e.getIndexes(ctx, atLeast8_0_13, strings.Contains(rest, "MariaDB"))
	if err != nil {
		return nil, err
	}

	// Query column info
	columnMap, err := e.getColumns(ctx, atLeast5_7_0)
	if err != nil {
		return nil, err
	}

	// Query check constraints
	checkMap, err := e.getCheckConstraints(ctx, atLeast8_0_16)
	if err != nil {
		return nil, err
	}

	// Query view info
	viewMap, err := e.getViews(ctx, columnMap)
	if err != nil {
		return nil, err
	}

	// Query triggers
	triggerMap, err := e.getTriggers(ctx)
	if err != nil {
		return nil, err
	}

	// Query events
	eventList, err := e.getEvents(ctx)
	if err != nil {
		return nil, err
	}
	schemaMetadata.Events = eventList

	// Query foreign keys
	foreignKeysMap, err := e.getForeignKeys(ctx)
	if err != nil {
		return nil, err
	}

	// Query partition info
	partitionTables, err := e.getPartitions(ctx)
	if err != nil {
		return nil, err
	}

	// Query functions and procedures
	functions, procedures, err := e.getRoutines(ctx)
	if err != nil {
		return nil, err
	}
	schemaMetadata.Functions = functions
	schemaMetadata.Procedures = procedures

	// Query table info
	tables, views, err := e.getTables(ctx, columnMap, indexMap, checkMap, foreignKeysMap, partitionTables, triggerMap, viewMap)
	if err != nil {
		return nil, err
	}
	schemaMetadata.Tables = tables
	schemaMetadata.Views = views

	// Query database info
	databaseMetadata := &schemaextract.DatabaseSchema{
		Name:    e.databaseName,
		Schemas: []*schemaextract.Schema{schemaMetadata},
	}

	databaseQuery := `
		SELECT
			DEFAULT_CHARACTER_SET_NAME,
			DEFAULT_COLLATION_NAME
		FROM information_schema.SCHEMATA
		WHERE SCHEMA_NAME = ?`
	if err := e.db.QueryRowContext(ctx, databaseQuery, e.databaseName).Scan(
		&databaseMetadata.CharacterSet,
		&databaseMetadata.Collation,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("database %q not found", e.databaseName)
		}
		return nil, err
	}

	return databaseMetadata, nil
}

// ListDatabases returns a list of all non-system databases.
func (e *Extractor) ListDatabases(ctx context.Context) ([]string, error) {
	systemDatabaseClause := makeSystemDatabaseClause()
	where := fmt.Sprintf("LOWER(SCHEMA_NAME) NOT IN (%s)", systemDatabaseClause)
	query := `
		SELECT SCHEMA_NAME
		FROM information_schema.SCHEMATA
		WHERE ` + where

	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			return nil, err
		}
		databases = append(databases, dbName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return databases, nil
}

func makeSystemDatabaseClause() string {
	var l []string
	for k := range systemDatabases {
		l = append(l, fmt.Sprintf("'%s'", k))
	}
	return strings.Join(l, ", ")
}

// getVersion returns the MySQL version string.
func (e *Extractor) getVersion(ctx context.Context) (string, string, error) {
	query := "SELECT VERSION()"
	var version string
	if err := e.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return "", "", fmt.Errorf("version query returned no rows")
		}
		return "", "", comparer.FormatErrorWithQuery(err, query)
	}

	// Version format: "8.0.30-0ubuntu0.20.04.2" or "5.7.39-log"
	parts := strings.Split(version, "-")
	if len(parts) == 0 {
		return "", "", fmt.Errorf("invalid version format: %s", version)
	}

	rest := ""
	if len(parts) > 1 {
		rest = strings.Join(parts[1:], "-")
	}

	return parts[0], rest, nil
}

// TableKey is used to map tables/views to their metadata.
type TableKey struct {
	Schema string
	Table  string
}

func (e *Extractor) getIndexes(ctx context.Context, atLeast8_0_13, isMariaDB bool) (map[TableKey]map[string]*schemaextract.Index, error) {
	indexMap := make(map[TableKey]map[string]*schemaextract.Index)

	indexQuery := `
		SELECT
			TABLE_NAME,
			INDEX_NAME,
			COLUMN_NAME,
			COLLATION,
			IFNULL(SUB_PART, -1),
			'',
			SEQ_IN_INDEX,
			INDEX_TYPE,
			NON_UNIQUE,
			1,
			INDEX_COMMENT
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX`

	if atLeast8_0_13 && !isMariaDB {
		indexQuery = `
			SELECT
				TABLE_NAME,
				INDEX_NAME,
				COLUMN_NAME,
				COLLATION,
				IFNULL(SUB_PART, -1),
				EXPRESSION,
				SEQ_IN_INDEX,
				INDEX_TYPE,
				NON_UNIQUE,
				CASE IS_VISIBLE WHEN 'YES' THEN 1 ELSE 0 END,
				INDEX_COMMENT
			FROM information_schema.STATISTICS
			WHERE TABLE_SCHEMA = ?
			ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX`
	}

	indexRows, err := e.db.QueryContext(ctx, indexQuery, e.databaseName)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, indexQuery)
	}
	defer indexRows.Close()

	for indexRows.Next() {
		var tableName, indexName, indexType, comment, expression string
		var columnName sql.NullString
		var expressionName sql.NullString
		var collation sql.NullString
		var position int
		var subPart int64
		var nonUnique, visible bool

		if err := indexRows.Scan(
			&tableName,
			&indexName,
			&columnName,
			&collation,
			&subPart,
			&expressionName,
			&position,
			&indexType,
			&nonUnique,
			&visible,
			&comment,
		); err != nil {
			return nil, err
		}

		if columnName.Valid {
			expression = columnName.String
		} else if expressionName.Valid {
			expression = fmt.Sprintf("(%s)", expressionName.String)
		}

		desc := false
		if collation.Valid && collation.String == "D" {
			desc = true
		}

		key := TableKey{Schema: "", Table: tableName}
		if _, ok := indexMap[key]; !ok {
			indexMap[key] = make(map[string]*schemaextract.Index)
		}
		if _, ok := indexMap[key][indexName]; !ok {
			indexMap[key][indexName] = &schemaextract.Index{
				Name:    indexName,
				Type:    indexType,
				Unique:  !nonUnique,
				Primary: indexName == "PRIMARY",
				Visible: visible,
				Comment: comment,
			}
		}
		indexMap[key][indexName].Expressions = append(indexMap[key][indexName].Expressions, expression)
		indexMap[key][indexName].KeyLength = append(indexMap[key][indexName].KeyLength, subPart)
		indexMap[key][indexName].Descending = append(indexMap[key][indexName].Descending, desc)
	}
	if err := indexRows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, indexQuery)
	}

	return indexMap, nil
}

func (e *Extractor) getColumns(ctx context.Context, atLeast5_7_0 bool) (map[TableKey][]*schemaextract.Column, error) {
	columnMap := make(map[TableKey][]*schemaextract.Column)

	columnQuery := `
		SELECT
			TABLE_NAME,
			IFNULL(COLUMN_NAME, ''),
			ORDINAL_POSITION,
			CASE WHEN COLUMN_DEFAULT is NULL THEN NULL ELSE QUOTE(COLUMN_DEFAULT) END,
			IS_NULLABLE,
			COLUMN_TYPE,
			IFNULL(CHARACTER_SET_NAME, ''),
			IFNULL(COLLATION_NAME, ''),
			QUOTE(COLUMN_COMMENT),
			convert(GENERATION_EXPRESSION using BINARY),
			EXTRA
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME, ORDINAL_POSITION`

	if !atLeast5_7_0 {
		columnQuery = `
		SELECT
			TABLE_NAME,
			IFNULL(COLUMN_NAME, ''),
			ORDINAL_POSITION,
			CASE WHEN COLUMN_DEFAULT is NULL THEN NULL ELSE QUOTE(COLUMN_DEFAULT) END,
			IS_NULLABLE,
			COLUMN_TYPE,
			IFNULL(CHARACTER_SET_NAME, ''),
			IFNULL(COLLATION_NAME, ''),
			QUOTE(COLUMN_COMMENT),
			NULL,
			EXTRA
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME, ORDINAL_POSITION`
	}

	columnRows, err := e.db.QueryContext(ctx, columnQuery, e.databaseName)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, columnQuery)
	}
	defer columnRows.Close()

	for columnRows.Next() {
		column := &schemaextract.Column{}
		var tableName, nullable, extra, tp string
		var defaultStr sql.NullString
		var generationExpr []byte

		if err := columnRows.Scan(
			&tableName,
			&column.Name,
			&column.Position,
			&defaultStr,
			&nullable,
			&tp,
			&column.CharacterSet,
			&column.Collation,
			&column.Comment,
			&generationExpr,
			&extra,
		); err != nil {
			return nil, err
		}

		column.Comment = stripSingleQuote(column.Comment)
		nullableBool, err := comparer.ConvertYesNo(nullable)
		if err != nil {
			return nil, err
		}
		column.Type = GetColumnTypeCanonicalSynonym(tp)
		column.Nullable = nullableBool
		setColumnMetadataDefault(column, defaultStr, nullableBool, extra)

		key := TableKey{Schema: "", Table: tableName}
		columnMap[key] = append(columnMap[key], column)

		// Handle generated columns
		invisible := containsInvisibleChars(generationExpr)
		iso88591Text, convertedErr := utf8ToISO88591(string(generationExpr))
		text := string(generationExpr)
		if invisible && convertedErr == nil {
			text = iso88591Text
		}
		text = strings.ReplaceAll(text, `\'`, `'`)

		if extra != "" && strings.Contains(strings.ToUpper(extra), virtualGenerated) && len(generationExpr) != 0 {
			column.Generation = &schemaextract.Generation{
				Type:       schemaextract.GenerationTypeVirtual,
				Expression: text,
			}
		} else if extra != "" && strings.Contains(strings.ToUpper(extra), storedGenerated) && len(generationExpr) != 0 {
			column.Generation = &schemaextract.Generation{
				Type:       schemaextract.GenerationTypeStored,
				Expression: text,
			}
		}
	}
	if err := columnRows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, columnQuery)
	}

	return columnMap, nil
}

func containsInvisibleChars(data []byte) bool {
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 {
			return true
		}
		if !unicode.IsPrint(r) {
			return true
		}
		data = data[size:]
	}
	return false
}

func utf8ToISO88591(utf8Str string) (string, error) {
	encoder := charmap.ISO8859_1.NewEncoder()
	isoBytes, _, err := transform.String(encoder, utf8Str)
	if err != nil {
		return "", err
	}
	return isoBytes, nil
}

func setColumnMetadataDefault(column *schemaextract.Column, defaultStr sql.NullString, nullableBool bool, extra string) {
	if defaultStr.Valid {
		unquotedDefault := unquoteMySQLString(defaultStr.String)
		switch {
		case isCurrentTimestampLike(unquotedDefault):
			column.Default = unquotedDefault
		case strings.Contains(extra, "DEFAULT_GENERATED"):
			unescapedDefault := unescapeExpressionDefault(unquotedDefault)
			column.Default = fmt.Sprintf("(%s)", unescapedDefault)
		default:
			column.Default = defaultStr.String
		}
	} else if strings.Contains(strings.ToUpper(extra), autoIncrementSymbol) {
		column.Default = autoIncrementSymbol
	} else if nullableBool {
		column.Default = "NULL"
	}

	if strings.Contains(extra, "on update CURRENT_TIMESTAMP") {
		re := regexp.MustCompile(`CURRENT_TIMESTAMP\((\d+)\)`)
		match := re.FindStringSubmatch(extra)
		if len(match) > 0 {
			digits := match[1]
			column.OnUpdate = fmt.Sprintf("CURRENT_TIMESTAMP(%s)", digits)
		} else {
			column.OnUpdate = "CURRENT_TIMESTAMP"
		}
	}
}

func unescapeExpressionDefault(s string) string {
	s = strings.ReplaceAll(s, `\'`, `'`)
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}

func isCurrentTimestampLike(s string) bool {
	upper := strings.ToUpper(s)
	return strings.HasPrefix(upper, "CURRENT_TIMESTAMP") || strings.HasPrefix(upper, "CURRENT_DATE")
}

func stripSingleQuote(s string) string {
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}

func unquoteMySQLString(s string) string {
	if len(s) == 0 {
		return s
	}

	s = stripSingleQuote(s)

	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case '\\':
				result = append(result, '\\')
				i++
			case '\'':
				result = append(result, '\'')
				i++
			case '0':
				result = append(result, 0)
				i++
			case 'Z':
				result = append(result, 0x1A)
				i++
			case 'n':
				result = append(result, '\n')
				i++
			case 't':
				result = append(result, '\t')
				i++
			case 'r':
				result = append(result, '\r')
				i++
			case 'b':
				result = append(result, '\b')
				i++
			default:
				result = append(result, s[i])
			}
		} else {
			result = append(result, s[i])
		}
	}
	return string(result)
}

func (e *Extractor) getCheckConstraints(ctx context.Context, atLeast8_0_16 bool) (map[TableKey][]*schemaextract.CheckConstraint, error) {
	checkMap := make(map[TableKey][]*schemaextract.CheckConstraint)
	if !atLeast8_0_16 {
		return checkMap, nil
	}

	checkQuery := `
		SELECT
			tc.TABLE_NAME,
			cc.CONSTRAINT_NAME,
			cc.CHECK_CLAUSE
		FROM information_schema.CHECK_CONSTRAINTS cc
			JOIN information_schema.TABLE_CONSTRAINTS tc ON cc.CONSTRAINT_NAME = tc.CONSTRAINT_NAME
		WHERE tc.CONSTRAINT_TYPE = 'CHECK' AND tc.TABLE_SCHEMA = ?`

	checkRows, err := e.db.QueryContext(ctx, checkQuery, e.databaseName)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, checkQuery)
	}
	defer checkRows.Close()

	for checkRows.Next() {
		check := &schemaextract.CheckConstraint{}
		var tableName string
		if err := checkRows.Scan(&tableName, &check.Name, &check.Expression); err != nil {
			return nil, err
		}
		key := TableKey{Schema: "", Table: tableName}
		checkMap[key] = append(checkMap[key], check)
	}
	if err := checkRows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, checkQuery)
	}

	return checkMap, nil
}

func (e *Extractor) getViews(ctx context.Context, columnMap map[TableKey][]*schemaextract.Column) (map[TableKey]*schemaextract.View, error) {
	viewMap := make(map[TableKey]*schemaextract.View)

	viewQuery := `
		SELECT
			TABLE_NAME,
			VIEW_DEFINITION
		FROM information_schema.VIEWS
		WHERE TABLE_SCHEMA = ?`

	viewRows, err := e.db.QueryContext(ctx, viewQuery, e.databaseName)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, viewQuery)
	}
	defer viewRows.Close()

	for viewRows.Next() {
		view := &schemaextract.View{}
		if err := viewRows.Scan(&view.Name, &view.Definition); err != nil {
			return nil, err
		}
		key := TableKey{Schema: "", Table: view.Name}
		view.Columns = columnMap[key]
		viewMap[key] = view
	}
	if err := viewRows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, viewQuery)
	}

	// Reconcile view definitions
	for key := range viewMap {
		def, err := e.reconcileViewDefinition(ctx, key.Table)
		if err != nil {
			return nil, err
		}
		if def != "" {
			viewMap[key].Definition = def
		}
	}

	return viewMap, nil
}

func (e *Extractor) reconcileViewDefinition(ctx context.Context, viewName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE VIEW `%s`.`%s`", e.databaseName, viewName)
	var createStmt, unused string
	if err := e.db.QueryRowContext(ctx, query).Scan(&unused, &createStmt, &unused, &unused); err != nil {
		if err == sql.ErrNoRows {
			slog.Warn("no rows return for query show create view", slog.String("viewName", viewName), slog.String("databaseName", e.databaseName))
			return "", nil
		}
		return "", fmt.Errorf("failed to scan row for query: %s: %w", query, err)
	}

	def, err := getViewDefFromCreateView(createStmt)
	if err != nil {
		slog.Warn("failed to get view definition", slog.String("viewName", viewName), slog.String("databaseName", e.databaseName), slog.Any("error", err))
		return "", nil
	}

	return def, nil
}

func getViewDefFromCreateView(createView string) (string, error) {
	viewDefMatching := viewDefMatcher.FindStringSubmatch(createView)
	if len(viewDefMatching) == 0 {
		return "", fmt.Errorf("failed to match view definition, %s", createView)
	}
	for i, name := range viewDefMatcher.SubexpNames() {
		if name == "def" && i < len(viewDefMatching) {
			return viewDefMatching[i], nil
		}
	}
	return "", fmt.Errorf("failed to match view definition, %s", createView)
}

func (e *Extractor) getTriggers(ctx context.Context) (map[TableKey][]*schemaextract.Trigger, error) {
	triggerMap := make(map[TableKey][]*schemaextract.Trigger)

	triggersQuery := `
	SELECT
		TRIGGER_NAME,
		EVENT_OBJECT_TABLE,
		EVENT_MANIPULATION,
		ACTION_TIMING,
		ACTION_STATEMENT,
		SQL_MODE,
		CHARACTER_SET_CLIENT,
		COLLATION_CONNECTION
	FROM INFORMATION_SCHEMA.TRIGGERS
	WHERE TRIGGER_SCHEMA = ?
	ORDER BY EVENT_OBJECT_TABLE ASC, EVENT_MANIPULATION ASC, ACTION_TIMING ASC, ACTION_ORDER ASC;`

	triggerRows, err := e.db.QueryContext(ctx, triggersQuery, e.databaseName)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, triggersQuery)
	}
	defer triggerRows.Close()

	for triggerRows.Next() {
		var name, table, event, timing, statement, sqlMode, charsetClient, collationConnection string
		if err := triggerRows.Scan(&name, &table, &event, &timing, &statement, &sqlMode, &charsetClient, &collationConnection); err != nil {
			return nil, err
		}
		trigger := &schemaextract.Trigger{
			Name:                name,
			Event:               event,
			Timing:              timing,
			Body:                statement,
			SQLMode:             sqlMode,
			CharacterSetClient:  charsetClient,
			CollationConnection: collationConnection,
		}
		tableKey := TableKey{Schema: "", Table: table}
		triggerMap[tableKey] = append(triggerMap[tableKey], trigger)
	}
	if err := triggerRows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, triggersQuery)
	}

	return triggerMap, nil
}

func (e *Extractor) getEvents(ctx context.Context) ([]*schemaextract.Event, error) {
	listEventsQuery := `
	SELECT
		EVENT_NAME,
		TIME_ZONE,
		SQL_MODE,
		CHARACTER_SET_CLIENT,
		COLLATION_CONNECTION
	FROM INFORMATION_SCHEMA.EVENTS
	WHERE EVENT_SCHEMA = ?
	ORDER BY EVENT_NAME ASC;`

	eventRows, err := e.db.QueryContext(ctx, listEventsQuery, e.databaseName)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, listEventsQuery)
	}
	defer eventRows.Close()

	var events []*schemaextract.Event
	for eventRows.Next() {
		var name, timeZone, sqlMode, charsetClient, collationConnection string
		if err := eventRows.Scan(&name, &timeZone, &sqlMode, &charsetClient, &collationConnection); err != nil {
			return nil, err
		}

		eventDef, err := e.getCreateEventStmt(ctx, name)
		if err != nil {
			return nil, err
		}

		event := &schemaextract.Event{
			Name:                name,
			TimeZone:            timeZone,
			Definition:          eventDef,
			SQLMode:             sqlMode,
			CharacterSetClient:  charsetClient,
			CollationConnection: collationConnection,
		}
		events = append(events, event)
	}
	if err := eventRows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, listEventsQuery)
	}

	return events, nil
}

func (e *Extractor) getCreateEventStmt(ctx context.Context, name string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE EVENT `%s`.`%s`", e.databaseName, name)
	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return "", comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var createEvent sql.NullString
	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}

	defIdx := -1
	for i, column := range columns {
		if strings.EqualFold(column, "Create Event") {
			defIdx = i
			break
		}
	}

	if defIdx == -1 {
		return "", fmt.Errorf("failed to find column Create Event")
	}

	for rows.Next() {
		dests := make([]any, len(columns))
		for i := 0; i < len(columns); i++ {
			if i == defIdx {
				dests[i] = &createEvent
			} else {
				dests[i] = new(string)
			}
		}

		if err := rows.Scan(dests...); err != nil {
			return "", err
		}
	}

	if err := rows.Err(); err != nil {
		return "", err
	}

	if createEvent.Valid {
		return createEvent.String, nil
	}
	return "", nil
}

func (e *Extractor) getForeignKeys(ctx context.Context) (map[TableKey][]*schemaextract.ForeignKey, error) {
	fkQuery := `
		SELECT
			TABLE_NAME,
			CONSTRAINT_NAME,
			REFERENCED_TABLE_NAME,
			DELETE_RULE,
			UPDATE_RULE,
			MATCH_OPTION
		FROM INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS
		WHERE LOWER(CONSTRAINT_SCHEMA) = ?;`

	kcuQuery := `
		SELECT
			TABLE_NAME,
			CONSTRAINT_NAME,
			COLUMN_NAME,
			REFERENCED_COLUMN_NAME
		FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
		WHERE POSITION_IN_UNIQUE_CONSTRAINT IS NOT NULL AND LOWER(CONSTRAINT_SCHEMA) = ?
		ORDER BY TABLE_NAME, CONSTRAINT_NAME, ORDINAL_POSITION;`

	type IndexKey struct {
		Schema string
		Table  string
		Index  string
	}

	fkRows, err := e.db.QueryContext(ctx, fkQuery, e.databaseName)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, fkQuery)
	}
	defer fkRows.Close()

	fkMap := make(map[IndexKey]*schemaextract.ForeignKey)
	for fkRows.Next() {
		var tableName string
		fk := &schemaextract.ForeignKey{}
		if err := fkRows.Scan(&tableName, &fk.Name, &fk.ReferencedTable, &fk.OnDelete, &fk.OnUpdate, &fk.MatchType); err != nil {
			return nil, err
		}
		key := IndexKey{Schema: "", Table: tableName, Index: fk.Name}
		fkMap[key] = fk
	}
	if err := fkRows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, fkQuery)
	}

	kcuQueryRows, err := e.db.QueryContext(ctx, kcuQuery, e.databaseName)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, kcuQuery)
	}
	defer kcuQueryRows.Close()

	for kcuQueryRows.Next() {
		var tableName, fkName, column, referencedColumn string
		if err := kcuQueryRows.Scan(&tableName, &fkName, &column, &referencedColumn); err != nil {
			return nil, err
		}
		key := IndexKey{Schema: "", Table: tableName, Index: fkName}
		if fk, ok := fkMap[key]; ok {
			fk.Columns = append(fk.Columns, column)
			fk.ReferencedColumns = append(fk.ReferencedColumns, referencedColumn)
		}
	}
	if err := kcuQueryRows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, kcuQuery)
	}

	result := make(map[TableKey][]*schemaextract.ForeignKey)
	for key, fk := range fkMap {
		tableKey := TableKey{Schema: "", Table: key.Table}
		result[tableKey] = append(result[tableKey], fk)
	}

	return result, nil
}

func (e *Extractor) getPartitions(ctx context.Context) (map[TableKey][]*schemaextract.Partition, error) {
	query := `
		SELECT
			TABLE_NAME,
			PARTITION_NAME,
			SUBPARTITION_NAME,
			PARTITION_METHOD,
			SUBPARTITION_METHOD,
			PARTITION_EXPRESSION,
			SUBPARTITION_EXPRESSION,
			PARTITION_DESCRIPTION
		FROM INFORMATION_SCHEMA.PARTITIONS
		WHERE TABLE_SCHEMA = ? AND PARTITION_NAME IS NOT NULL
		ORDER BY TABLE_NAME ASC, PARTITION_ORDINAL_POSITION ASC, SUBPARTITION_ORDINAL_POSITION ASC;`

	rows, err := e.db.QueryContext(ctx, query, e.databaseName)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	type partitionKey struct {
		tableName     string
		partitionName string
	}

	partitionMap := make(map[partitionKey]int)
	result := make(map[TableKey][]*schemaextract.Partition)

	for rows.Next() {
		var tableName, partitionName, partitionMethod string
		var subpartitionName, subpartitionMethod, subpartitionExpression, partitionExpression, partitionDescription sql.NullString

		if err := rows.Scan(&tableName, &partitionName, &subpartitionName, &partitionMethod, &subpartitionMethod, &partitionExpression, &subpartitionExpression, &partitionDescription); err != nil {
			return nil, err
		}

		pkey := partitionKey{tableName: tableName, partitionName: partitionName}
		tableKey := TableKey{Schema: "", Table: tableName}

		if _, ok := partitionMap[pkey]; !ok {
			tp := convertToPartitionType(partitionMethod)
			expression := ""
			if partitionExpression.Valid {
				expression = partitionExpression.String
			}

			value := ""
			if partitionDescription.Valid {
				value = partitionDescription.String
			}

			partition := &schemaextract.Partition{
				Name:          partitionName,
				Type:          tp,
				Expression:    expression,
				Value:         value,
				Subpartitions: []*schemaextract.Partition{},
			}
			partitionMap[pkey] = len(result[tableKey])
			result[tableKey] = append(result[tableKey], partition)
		}

		if subpartitionName.Valid {
			tp := convertToPartitionType(subpartitionMethod.String)
			expression := ""
			if subpartitionExpression.Valid {
				expression = subpartitionExpression.String
			}

			subPartition := &schemaextract.Partition{
				Name:          subpartitionName.String,
				Type:          tp,
				Expression:    expression,
				Value:         "",
				Subpartitions: []*schemaextract.Partition{},
			}

			if idx, ok := partitionMap[pkey]; ok {
				result[tableKey][idx].Subpartitions = append(result[tableKey][idx].Subpartitions, subPartition)
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Get default partition counts using SHOW CREATE TABLE
	for tableKey, partitions := range result {
		if len(partitions) == 0 {
			continue
		}
		showQuery := fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`", e.databaseName, tableKey.Table)
		showRows, err := e.db.QueryContext(ctx, showQuery)
		if err != nil {
			slog.Warn("failed to execute query", slog.String("query", showQuery), slog.Any("error", err))
			continue
		}

		for showRows.Next() {
			var tableName, createTable string
			if err := showRows.Scan(&tableName, &createTable); err != nil {
				slog.Warn("failed to scan row", slog.String("query", showQuery), slog.Any("error", err))
				continue
			}

			partitionRegexp := regexp.MustCompile(`[^B]PARTITIONS (?P<partitionNum>\d+)`)
			subPartitionRegexp := regexp.MustCompile(`SUBPARTITIONS (?P<subPartitionNum>\d+)`)

			partitionNum := 0
			subPartitionNum := 0

			if partitionRegexp.MatchString(createTable) {
				partitionNum, _ = strconv.Atoi(partitionRegexp.FindStringSubmatch(createTable)[1])
			}
			if subPartitionRegexp.MatchString(createTable) {
				subPartitionNum, _ = strconv.Atoi(subPartitionRegexp.FindStringSubmatch(createTable)[1])
			}

			for _, partition := range partitions {
				if partitionNum != 0 {
					partition.UseDefault = strconv.Itoa(partitionNum)
				}
				for _, subPartition := range partition.Subpartitions {
					if subPartitionNum != 0 {
						subPartition.UseDefault = strconv.Itoa(subPartitionNum)
					}
				}
			}
		}
		showRows.Close()
	}

	return result, nil
}

func convertToPartitionType(tp string) schemaextract.PartitionType {
	switch strings.ToUpper(tp) {
	case "RANGE":
		return schemaextract.PartitionTypeRange
	case "RANGE COLUMNS":
		return schemaextract.PartitionTypeRangeColumns
	case "LIST":
		return schemaextract.PartitionTypeList
	case "LIST COLUMNS":
		return schemaextract.PartitionTypeListColumns
	case "HASH":
		return schemaextract.PartitionTypeHash
	case "KEY":
		return schemaextract.PartitionTypeKey
	case "LINEAR HASH":
		return schemaextract.PartitionTypeLinearHash
	case "LINEAR KEY":
		return schemaextract.PartitionTypeLinearKey
	default:
		return ""
	}
}

func (e *Extractor) getRoutines(ctx context.Context) ([]*schemaextract.Function, []*schemaextract.Procedure, error) {
	routinesQuery := `
		SELECT
			ROUTINE_NAME,
			ROUTINE_TYPE,
			SQL_MODE,
			CHARACTER_SET_CLIENT,
			COLLATION_CONNECTION,
			DATABASE_COLLATION,
			IFNULL(ROUTINE_COMMENT, '')
		FROM
			INFORMATION_SCHEMA.ROUTINES
		WHERE ROUTINE_SCHEMA = ? AND ROUTINE_TYPE IN ('FUNCTION', 'PROCEDURE')
		ORDER BY ROUTINE_TYPE, ROUTINE_NAME;`

	routineRows, err := e.db.QueryContext(ctx, routinesQuery, e.databaseName)
	if err != nil {
		return nil, nil, comparer.FormatErrorWithQuery(err, routinesQuery)
	}
	defer routineRows.Close()

	var functions []*schemaextract.Function
	var procedures []*schemaextract.Procedure

	for routineRows.Next() {
		var name, routineType, routineComment string
		var sqlMode, charsetClient, collationConnection, databaseCollation sql.NullString

		if err := routineRows.Scan(&name, &routineType, &sqlMode, &charsetClient, &collationConnection, &databaseCollation, &routineComment); err != nil {
			return nil, nil, err
		}

		if strings.EqualFold(routineType, "PROCEDURE") {
			procedureDef, err := e.getCreateProcedureStmt(ctx, name)
			if err != nil {
				return nil, nil, err
			}
			procedures = append(procedures, &schemaextract.Procedure{
				Name:                name,
				Definition:          procedureDef,
				SQLMode:             sqlMode.String,
				CharacterSetClient:  charsetClient.String,
				CollationConnection: collationConnection.String,
				DatabaseCollation:   databaseCollation.String,
				Comment:             routineComment,
			})
		} else {
			functionDef, err := e.getCreateFunctionStmt(ctx, name)
			if err != nil {
				return nil, nil, err
			}
			functions = append(functions, &schemaextract.Function{
				Name:                name,
				Definition:          functionDef,
				SQLMode:             sqlMode.String,
				CharacterSetClient:  charsetClient.String,
				CollationConnection: collationConnection.String,
				DatabaseCollation:   databaseCollation.String,
				Comment:             routineComment,
			})
		}
	}
	if err := routineRows.Err(); err != nil {
		return nil, nil, comparer.FormatErrorWithQuery(err, routinesQuery)
	}

	return functions, procedures, nil
}

func (e *Extractor) getCreateFunctionStmt(ctx context.Context, functionName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE FUNCTION `%s`.`%s`", e.databaseName, functionName)
	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return "", comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var createFunction sql.NullString
	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}

	defIdx := -1
	for i, column := range columns {
		if strings.EqualFold(column, "Create Function") {
			defIdx = i
			break
		}
	}

	if defIdx == -1 {
		return "", fmt.Errorf("failed to find column Create Function")
	}

	for rows.Next() {
		dests := make([]any, len(columns))
		for i := 0; i < len(columns); i++ {
			if i == defIdx {
				dests[i] = &createFunction
			} else {
				dests[i] = new(string)
			}
		}

		if err := rows.Scan(dests...); err != nil {
			return "", err
		}
	}

	if err := rows.Err(); err != nil {
		return "", err
	}

	if createFunction.Valid {
		f := createFunction.String

		functionSymbolIdx := strings.Index(f, " FUNCTION ")
		if functionSymbolIdx >= 0 {
			f = fmt.Sprintf("CREATE%s", f[functionSymbolIdx:])
		}

		if charsetIdx := strings.Index(f, " CHARSET "); charsetIdx != -1 {
			if newLineIdx := strings.Index(f, "\n"); newLineIdx != -1 {
				f = f[:charsetIdx] + f[newLineIdx:]
			}
		}

		return f, nil
	}
	return "", nil
}

func (e *Extractor) getCreateProcedureStmt(ctx context.Context, procedureName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE PROCEDURE `%s`.`%s`", e.databaseName, procedureName)
	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return "", comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var createProcedure sql.NullString
	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}

	defIdx := -1
	for i, column := range columns {
		if strings.EqualFold(column, "Create Procedure") {
			defIdx = i
			break
		}
	}

	if defIdx == -1 {
		return "", fmt.Errorf("failed to find column Create Procedure")
	}

	for rows.Next() {
		dests := make([]any, len(columns))
		for i := 0; i < len(columns); i++ {
			if i == defIdx {
				dests[i] = &createProcedure
			} else {
				dests[i] = new(string)
			}
		}

		if err := rows.Scan(dests...); err != nil {
			return "", err
		}
	}

	if err := rows.Err(); err != nil {
		return "", err
	}

	if createProcedure.Valid {
		p := createProcedure.String

		procedureSymbolIdx := strings.Index(p, " PROCEDURE ")
		if procedureSymbolIdx >= 0 {
			p = fmt.Sprintf("CREATE%s", p[procedureSymbolIdx:])
		}

		return p, nil
	}
	return "", nil
}

func (e *Extractor) getTables(
	ctx context.Context,
	columnMap map[TableKey][]*schemaextract.Column,
	indexMap map[TableKey]map[string]*schemaextract.Index,
	checkMap map[TableKey][]*schemaextract.CheckConstraint,
	foreignKeysMap map[TableKey][]*schemaextract.ForeignKey,
	partitionTables map[TableKey][]*schemaextract.Partition,
	triggerMap map[TableKey][]*schemaextract.Trigger,
	viewMap map[TableKey]*schemaextract.View,
) ([]*schemaextract.Table, []*schemaextract.View, error) {
	tableQuery := `
		SELECT
			TABLES.TABLE_NAME,
			TABLES.TABLE_TYPE,
			IFNULL(TABLES.ENGINE, ''),
			IFNULL(TABLES.TABLE_COLLATION, ''),
			IFNULL(TABLES.TABLE_ROWS, 0),
			IFNULL(TABLES.DATA_LENGTH, 0),
			IFNULL(TABLES.INDEX_LENGTH, 0),
			IFNULL(TABLES.DATA_FREE, 0),
			IFNULL(TABLES.CREATE_OPTIONS, ''),
			QUOTE(IFNULL(TABLES.TABLE_COMMENT, '')),
			IFNULL(CCSA.CHARACTER_SET_NAME, '')
		FROM information_schema.TABLES TABLES
		LEFT JOIN information_schema.COLLATION_CHARACTER_SET_APPLICABILITY CCSA
		ON TABLES.TABLE_COLLATION = CCSA.COLLATION_NAME
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME`

	tableRows, err := e.db.QueryContext(ctx, tableQuery, e.databaseName)
	if err != nil {
		return nil, nil, comparer.FormatErrorWithQuery(err, tableQuery)
	}
	defer tableRows.Close()

	var tables []*schemaextract.Table
	var views []*schemaextract.View

	for tableRows.Next() {
		var tableName, tableType, engine, collation, createOptions, comment, charset string
		var rowCount, dataSize, indexSize, dataFree int64

		if err := tableRows.Scan(&tableName, &tableType, &engine, &collation, &rowCount, &dataSize, &indexSize, &dataFree, &createOptions, &comment, &charset); err != nil {
			return nil, nil, err
		}

		comment = stripSingleQuote(comment)
		key := TableKey{Schema: "", Table: tableName}

		switch tableType {
		case baseTableType:
			table := &schemaextract.Table{
				Name:             tableName,
				Columns:          columnMap[key],
				ForeignKeys:      foreignKeysMap[key],
				Engine:           engine,
				Collation:        collation,
				RowCount:         rowCount,
				DataSize:         dataSize,
				IndexSize:        indexSize,
				DataFree:         dataFree,
				CreateOptions:    createOptions,
				Comment:          comment,
				Partitions:       partitionTables[key],
				CheckConstraints: checkMap[key],
				Charset:          charset,
				Triggers:         triggerMap[key],
			}

			// Add indexes
			if indexes, ok := indexMap[key]; ok {
				for _, index := range indexes {
					table.Indexes = append(table.Indexes, index)
				}
			}

			tables = append(tables, table)
		case viewTableType:
			if view, ok := viewMap[key]; ok {
				views = append(views, view)
			}
		}
	}
	if err := tableRows.Err(); err != nil {
		return nil, nil, comparer.FormatErrorWithQuery(err, tableQuery)
	}

	return tables, views, nil
}
