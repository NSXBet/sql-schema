package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/lib/pq"

	"github.com/nsxbet/sql-schema"
	"github.com/nsxbet/sql-schema/comparer"
)

// Extractor extracts PostgreSQL database schema.
type Extractor struct {
	db           *sql.DB
	databaseName string
}

// NewExtractor creates a new PostgreSQL schema extractor.
func NewExtractor(db *sql.DB, databaseName string) *Extractor {
	return &Extractor{
		db:           db,
		databaseName: databaseName,
	}
}

// ExtractSchema extracts the complete schema metadata for the database.
func (e *Extractor) ExtractSchema(ctx context.Context) (*schemaextract.DatabaseSchema, error) {
	// Get PostgreSQL version
	version, err := e.getVersion(ctx)
	if err != nil {
		return nil, err
	}

	isAtLeastPG10 := e.isAtLeastPG10(version)

	// Get search path
	searchPath, err := e.getSearchPath(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get search path for database %q: %w", e.databaseName, err)
	}

	// Begin transaction for consistent snapshot
	txn, err := e.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	// Set search path to empty for consistent schema references
	if err := e.setTxSearchPath(txn, ""); err != nil {
		return nil, fmt.Errorf("failed to set search path: %w", err)
	}

	// Get extension dependencies
	extensionDepend, err := e.getExtensionDepend(txn)
	if err != nil {
		return nil, fmt.Errorf("failed to get extension dependencies: %w", err)
	}

	// Get schemas
	schemaMap, err := e.getSchemas(txn)
	if err != nil {
		return nil, fmt.Errorf("failed to get schemas: %w", err)
	}

	// Get columns
	columnMap, err := e.getTableColumns(txn)
	if err != nil {
		return nil, fmt.Errorf("failed to get table columns: %w", err)
	}

	// Get indexes
	indexMap, err := e.getIndexes(txn)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}

	// Get triggers
	triggerMap, err := e.getTriggers(txn)
	if err != nil {
		return nil, fmt.Errorf("failed to get triggers: %w", err)
	}

	// Get check constraints
	checkMap, err := e.getChecks(txn)
	if err != nil {
		return nil, fmt.Errorf("failed to get check constraints: %w", err)
	}

	// Get foreign keys
	foreignKeyMap, err := e.getForeignKeys(txn)
	if err != nil {
		return nil, fmt.Errorf("failed to get foreign keys: %w", err)
	}

	// Get partitions (PostgreSQL 10+)
	var partitionMap map[TableKey][]*schemaextract.Partition
	if isAtLeastPG10 {
		partitionMap, err = e.getTablePartitions(txn)
		if err != nil {
			return nil, fmt.Errorf("failed to get table partitions: %w", err)
		}
	}

	// Get tables
	tableMap, err := e.getTables(txn, columnMap, indexMap, triggerMap, checkMap, foreignKeyMap, partitionMap)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	// Get views
	viewMap, err := e.getViews(txn, columnMap)
	if err != nil {
		return nil, fmt.Errorf("failed to get views: %w", err)
	}

	// Get materialized views
	materializedViewMap, err := e.getMaterializedViews(txn, columnMap)
	if err != nil {
		return nil, fmt.Errorf("failed to get materialized views: %w", err)
	}

	// Get functions
	functionMap, err := e.getFunctions(txn, extensionDepend)
	if err != nil {
		return nil, fmt.Errorf("failed to get functions: %w", err)
	}

	// Get procedures (PostgreSQL 11+)
	procedureMap, err := e.getProcedures(txn, extensionDepend)
	if err != nil {
		return nil, fmt.Errorf("failed to get procedures: %w", err)
	}

	// Get sequences
	sequenceMap, err := e.getSequences(txn)
	if err != nil {
		return nil, fmt.Errorf("failed to get sequences: %w", err)
	}

	// Get extensions
	extensionMap, err := e.getExtensions(txn)
	if err != nil {
		return nil, fmt.Errorf("failed to get extensions: %w", err)
	}

	// Get enum types
	enumMap, err := e.getEnumTypes(txn)
	if err != nil {
		return nil, fmt.Errorf("failed to get enum types: %w", err)
	}

	// Commit transaction
	if err := txn.Commit(); err != nil {
		return nil, err
	}

	// Build schema metadata
	var schemas []*schemaextract.Schema
	for schemaName, schemaInfo := range schemaMap {
		if IsSystemSchema(schemaName) {
			continue
		}

		schema := &schemaextract.Schema{
			Name:              schemaName,
			Tables:            tableMap[schemaName],
			Views:             viewMap[schemaName],
			MaterializedViews: materializedViewMap[schemaName],
			Functions:         functionMap[schemaName],
			Procedures:        procedureMap[schemaName],
			Sequences:         sequenceMap[schemaName],
			Extensions:        extensionMap[schemaName],
			EnumTypes:         enumMap[schemaName],
		}

		// Set schema owner from schemaInfo if needed
		_ = schemaInfo // For future use

		schemas = append(schemas, schema)
	}

	databaseMetadata := &schemaextract.DatabaseSchema{
		Name:       e.databaseName,
		SearchPath: searchPath,
		Schemas:    schemas,
	}

	return databaseMetadata, nil
}

// ListDatabases returns a list of all non-system databases.
func (e *Extractor) ListDatabases(ctx context.Context) ([]string, error) {
	query := `
		SELECT datname
		FROM pg_database
		WHERE datistemplate = false`

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
		if !IsSystemDatabase(dbName) {
			databases = append(databases, dbName)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return databases, nil
}

func (e *Extractor) getVersion(ctx context.Context) (string, error) {
	query := "SHOW server_version"
	var version string
	if err := e.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("version query returned no rows")
		}
		return "", comparer.FormatErrorWithQuery(err, query)
	}
	return version, nil
}

func (e *Extractor) isAtLeastPG10(version string) bool {
	// Version format: "14.5 (Debian 14.5-1.pgdg110+1)"
	parts := strings.Fields(version)
	if len(parts) == 0 {
		return false
	}
	semVersion, err := semver.Make(parts[0] + ".0")
	if err != nil {
		return false
	}
	return semVersion.GE(semver.MustParse("10.0.0"))
}

func (e *Extractor) getSearchPath(ctx context.Context) ([]string, error) {
	query := "SHOW search_path"
	var searchPathStr string
	if err := e.db.QueryRowContext(ctx, query).Scan(&searchPathStr); err != nil {
		if err == sql.ErrNoRows {
			return []string{"public"}, nil
		}
		return nil, comparer.FormatErrorWithQuery(err, query)
	}

	// Parse search path: "$user", public
	var searchPath []string
	parts := strings.Split(searchPathStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"`)
		if part != "" {
			searchPath = append(searchPath, part)
		}
	}

	return searchPath, nil
}

func (e *Extractor) setTxSearchPath(txn *sql.Tx, searchPath string) error {
	query := fmt.Sprintf("SET search_path TO %s", searchPath)
	if searchPath == "" {
		query = "SET search_path TO ''"
	}
	if _, err := txn.Exec(query); err != nil {
		return comparer.FormatErrorWithQuery(err, query)
	}
	return nil
}

// TableKey is used to map tables/views to their metadata.
type TableKey struct {
	Schema string
	Table  string
}

type schemaInfo struct {
	name    string
	owner   string
	comment string
}

func (e *Extractor) getSchemas(txn *sql.Tx) (map[string]*schemaInfo, error) {
	query := `
		SELECT
			nspname,
			pg_catalog.pg_get_userbyid(nspowner),
			pg_catalog.obj_description(oid, 'pg_namespace')
		FROM pg_catalog.pg_namespace
		WHERE nspname NOT IN (` + SystemSchemaWhereClause + `)`

	rows, err := txn.Query(query)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	schemaMap := make(map[string]*schemaInfo)
	for rows.Next() {
		var name, owner string
		var comment sql.NullString
		if err := rows.Scan(&name, &owner, &comment); err != nil {
			return nil, err
		}
		schemaMap[name] = &schemaInfo{
			name:    name,
			owner:   owner,
			comment: comment.String,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}

	return schemaMap, nil
}

type columnOidKey struct {
	tableOid int
	position int
}

func (e *Extractor) getTableColumns(txn *sql.Tx) (map[TableKey][]*schemaextract.Column, error) {
	query := `
		SELECT
			c.oid,
			n.nspname,
			c.relname,
			a.attnum,
			a.attname,
			pg_catalog.format_type(a.atttypid, a.atttypmod),
			NOT a.attnotnull,
			pg_catalog.pg_get_expr(ad.adbin, ad.adrelid),
			CASE WHEN a.attgenerated = 's' THEN pg_catalog.pg_get_expr(ad.adbin, ad.adrelid) ELSE '' END,
			a.attgenerated,
			col_description(c.oid, a.attnum)
		FROM pg_catalog.pg_class c
			LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
			LEFT JOIN pg_catalog.pg_attribute a ON c.oid = a.attrelid
			LEFT JOIN pg_catalog.pg_attrdef ad ON a.attrelid = ad.adrelid AND a.attnum = ad.adnum
		WHERE c.relkind IN ('r', 'v', 'm', 'f', 'p')
			AND a.attnum > 0
			AND NOT a.attisdropped
			AND n.nspname NOT IN (` + SystemSchemaWhereClause + `)
		ORDER BY n.nspname, c.relname, a.attnum`

	rows, err := txn.Query(query)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	columnMap := make(map[TableKey][]*schemaextract.Column)

	for rows.Next() {
		var tableOid, position int
		var schemaName, tableName, columnName, dataType string
		var nullable bool
		var defaultValue, generatedExpr, generated, comment sql.NullString

		if err := rows.Scan(&tableOid, &schemaName, &tableName, &position, &columnName, &dataType, &nullable, &defaultValue, &generatedExpr, &generated, &comment); err != nil {
			return nil, err
		}

		column := &schemaextract.Column{
			Name:     columnName,
			Position: int32(position),
			Type:     dataType,
			Nullable: nullable,
			Comment:  comment.String,
		}

		if defaultValue.Valid {
			column.Default = defaultValue.String
		}

		if generated.Valid && generated.String != "" {
			genType := schemaextract.GenerationTypeStored
			if generated.String == "v" {
				genType = schemaextract.GenerationTypeVirtual
			}
			column.Generation = &schemaextract.Generation{
				Type:       genType,
				Expression: generatedExpr.String,
			}
		}

		key := TableKey{Schema: schemaName, Table: tableName}
		columnMap[key] = append(columnMap[key], column)
	}
	if err := rows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}

	return columnMap, nil
}

func (e *Extractor) getIndexes(txn *sql.Tx) (map[TableKey][]*schemaextract.Index, error) {
	query := `
		SELECT
			n.nspname,
			c.relname,
			i.relname,
			idx.indisprimary,
			idx.indisunique,
			pg_catalog.pg_get_indexdef(idx.indexrelid),
			obj_description(i.oid, 'pg_class')
		FROM pg_catalog.pg_index idx
			JOIN pg_catalog.pg_class c ON c.oid = idx.indrelid
			JOIN pg_catalog.pg_class i ON i.oid = idx.indexrelid
			JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname NOT IN (` + SystemSchemaWhereClause + `)
		ORDER BY n.nspname, c.relname, i.relname`

	rows, err := txn.Query(query)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	indexMap := make(map[TableKey][]*schemaextract.Index)
	indexDefRe := regexp.MustCompile(`USING\s+(\w+)\s+\(([^)]+)\)`)

	for rows.Next() {
		var schemaName, tableName, indexName, indexDef string
		var isPrimary, isUnique bool
		var comment sql.NullString

		if err := rows.Scan(&schemaName, &tableName, &indexName, &isPrimary, &isUnique, &indexDef, &comment); err != nil {
			return nil, err
		}

		// Parse index definition to get type and expressions
		indexType := "BTREE"
		var expressions []string

		matches := indexDefRe.FindStringSubmatch(indexDef)
		if len(matches) >= 3 {
			indexType = strings.ToUpper(matches[1])
			exprStr := matches[2]
			// Split by comma, but handle function calls
			expressions = splitIndexExpressions(exprStr)
		}

		index := &schemaextract.Index{
			Name:        indexName,
			Type:        indexType,
			Expressions: expressions,
			Primary:     isPrimary,
			Unique:      isUnique,
			Visible:     true,
			Comment:     comment.String,
		}

		key := TableKey{Schema: schemaName, Table: tableName}
		indexMap[key] = append(indexMap[key], index)
	}
	if err := rows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}

	return indexMap, nil
}

func splitIndexExpressions(exprStr string) []string {
	var expressions []string
	var current strings.Builder
	depth := 0

	for _, ch := range exprStr {
		switch ch {
		case '(':
			depth++
			current.WriteRune(ch)
		case ')':
			depth--
			current.WriteRune(ch)
		case ',':
			if depth == 0 {
				expr := strings.TrimSpace(current.String())
				if expr != "" {
					expressions = append(expressions, expr)
				}
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	// Add last expression
	expr := strings.TrimSpace(current.String())
	if expr != "" {
		expressions = append(expressions, expr)
	}

	return expressions
}

func (e *Extractor) getTriggers(txn *sql.Tx) (map[TableKey][]*schemaextract.Trigger, error) {
	query := `
		SELECT
			n.nspname,
			c.relname,
			t.tgname,
			CASE
				WHEN t.tgtype & 2 = 2 THEN 'BEFORE'
				WHEN t.tgtype & 64 = 64 THEN 'INSTEAD OF'
				ELSE 'AFTER'
			END,
			CASE
				WHEN t.tgtype & 4 = 4 THEN 'INSERT'
				WHEN t.tgtype & 8 = 8 THEN 'DELETE'
				WHEN t.tgtype & 16 = 16 THEN 'UPDATE'
				ELSE 'UNKNOWN'
			END,
			pg_catalog.pg_get_triggerdef(t.oid),
			obj_description(t.oid, 'pg_trigger')
		FROM pg_catalog.pg_trigger t
			JOIN pg_catalog.pg_class c ON c.oid = t.tgrelid
			JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE NOT t.tgisinternal
			AND n.nspname NOT IN (` + SystemSchemaWhereClause + `)
		ORDER BY n.nspname, c.relname, t.tgname`

	rows, err := txn.Query(query)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	triggerMap := make(map[TableKey][]*schemaextract.Trigger)

	for rows.Next() {
		var schemaName, tableName, triggerName, timing, event, body string
		var comment sql.NullString

		if err := rows.Scan(&schemaName, &tableName, &triggerName, &timing, &event, &body, &comment); err != nil {
			return nil, err
		}

		trigger := &schemaextract.Trigger{
			Name:    triggerName,
			Timing:  timing,
			Event:   event,
			Body:    body,
			Comment: comment.String,
		}

		key := TableKey{Schema: schemaName, Table: tableName}
		triggerMap[key] = append(triggerMap[key], trigger)
	}
	if err := rows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}

	return triggerMap, nil
}

func (e *Extractor) getChecks(txn *sql.Tx) (map[TableKey][]*schemaextract.CheckConstraint, error) {
	query := `
		SELECT
			n.nspname,
			c.relname,
			con.conname,
			pg_catalog.pg_get_constraintdef(con.oid)
		FROM pg_catalog.pg_constraint con
			JOIN pg_catalog.pg_class c ON c.oid = con.conrelid
			JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE con.contype = 'c'
			AND n.nspname NOT IN (` + SystemSchemaWhereClause + `)
		ORDER BY n.nspname, c.relname, con.conname`

	rows, err := txn.Query(query)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	checkMap := make(map[TableKey][]*schemaextract.CheckConstraint)

	for rows.Next() {
		var schemaName, tableName, checkName, checkDef string

		if err := rows.Scan(&schemaName, &tableName, &checkName, &checkDef); err != nil {
			return nil, err
		}

		// Remove "CHECK " prefix from definition
		expression := strings.TrimSpace(strings.TrimPrefix(checkDef, "CHECK "))
		expression = strings.TrimPrefix(expression, "(")
		expression = strings.TrimSuffix(expression, ")")

		check := &schemaextract.CheckConstraint{
			Name:       checkName,
			Expression: expression,
		}

		key := TableKey{Schema: schemaName, Table: tableName}
		checkMap[key] = append(checkMap[key], check)
	}
	if err := rows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}

	return checkMap, nil
}

func (e *Extractor) getForeignKeys(txn *sql.Tx) (map[TableKey][]*schemaextract.ForeignKey, error) {
	query := `
		SELECT
			n.nspname,
			c.relname,
			con.conname,
			pg_catalog.pg_get_constraintdef(con.oid),
			con.confupdtype,
			con.confdeltype
		FROM pg_catalog.pg_constraint con
			JOIN pg_catalog.pg_class c ON c.oid = con.conrelid
			JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE con.contype = 'f'
			AND n.nspname NOT IN (` + SystemSchemaWhereClause + `)
		ORDER BY n.nspname, c.relname, con.conname`

	rows, err := txn.Query(query)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	foreignKeyMap := make(map[TableKey][]*schemaextract.ForeignKey)
	fkDefRe := regexp.MustCompile(`FOREIGN KEY \(([^)]+)\) REFERENCES\s+(?:(\w+)\.)?(\w+)\s*\(([^)]+)\)`)

	for rows.Next() {
		var schemaName, tableName, fkName, fkDef, updateType, deleteType string

		if err := rows.Scan(&schemaName, &tableName, &fkName, &fkDef, &updateType, &deleteType); err != nil {
			return nil, err
		}

		// Parse FK definition
		matches := fkDefRe.FindStringSubmatch(fkDef)
		if len(matches) < 5 {
			slog.Warn("failed to parse foreign key definition", slog.String("definition", fkDef))
			continue
		}

		columnsStr := matches[1]
		refSchema := matches[2]
		refTable := matches[3]
		refColumnsStr := matches[4]

		// Parse columns
		columns := parseColumnList(columnsStr)
		refColumns := parseColumnList(refColumnsStr)

		// Prepend schema to table name if specified
		if refSchema != "" {
			refTable = refSchema + "." + refTable
		}

		// Convert action types
		onUpdate := convertFKAction(updateType)
		onDelete := convertFKAction(deleteType)

		fk := &schemaextract.ForeignKey{
			Name:              fkName,
			Columns:           columns,
			ReferencedTable:   refTable,
			ReferencedColumns: refColumns,
			OnUpdate:          onUpdate,
			OnDelete:          onDelete,
		}

		key := TableKey{Schema: schemaName, Table: tableName}
		foreignKeyMap[key] = append(foreignKeyMap[key], fk)
	}
	if err := rows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}

	return foreignKeyMap, nil
}

func parseColumnList(columnsStr string) []string {
	var columns []string
	parts := strings.Split(columnsStr, ",")
	for _, part := range parts {
		col := strings.TrimSpace(part)
		col = strings.Trim(col, `"`)
		if col != "" {
			columns = append(columns, col)
		}
	}
	return columns
}

func convertFKAction(actionType string) string {
	switch actionType {
	case "a":
		return "NO ACTION"
	case "r":
		return "RESTRICT"
	case "c":
		return "CASCADE"
	case "n":
		return "SET NULL"
	case "d":
		return "SET DEFAULT"
	default:
		return "NO ACTION"
	}
}

func (e *Extractor) getTablePartitions(txn *sql.Tx) (map[TableKey][]*schemaextract.Partition, error) {
	query := `
		SELECT
			n.nspname,
			c.relname,
			p.relname,
			pg_catalog.pg_get_expr(c.relpartbound, c.oid)
		FROM pg_catalog.pg_class c
			JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
			JOIN pg_catalog.pg_inherits i ON i.inhrelid = c.oid
			JOIN pg_catalog.pg_class p ON p.oid = i.inhparent
		WHERE c.relispartition
			AND n.nspname NOT IN (` + SystemSchemaWhereClause + `)
		ORDER BY n.nspname, p.relname, c.relname`

	rows, err := txn.Query(query)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	partitionMap := make(map[TableKey][]*schemaextract.Partition)

	for rows.Next() {
		var schemaName, tableName, parentName, partitionExpr string

		if err := rows.Scan(&schemaName, &tableName, &parentName, &partitionExpr); err != nil {
			return nil, err
		}

		partition := &schemaextract.Partition{
			Name:       tableName,
			Type:       "RANGE", // PostgreSQL supports RANGE, LIST, HASH
			Expression: partitionExpr,
		}

		// The partition belongs to the parent table
		key := TableKey{Schema: schemaName, Table: parentName}
		partitionMap[key] = append(partitionMap[key], partition)
	}
	if err := rows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}

	return partitionMap, nil
}

func (e *Extractor) getTables(
	txn *sql.Tx,
	columnMap map[TableKey][]*schemaextract.Column,
	indexMap map[TableKey][]*schemaextract.Index,
	triggerMap map[TableKey][]*schemaextract.Trigger,
	checkMap map[TableKey][]*schemaextract.CheckConstraint,
	foreignKeyMap map[TableKey][]*schemaextract.ForeignKey,
	partitionMap map[TableKey][]*schemaextract.Partition,
) (map[string][]*schemaextract.Table, error) {
	query := `
		SELECT
			n.nspname,
			c.relname,
			pg_catalog.obj_description(c.oid, 'pg_class'),
			pg_catalog.pg_total_relation_size(c.oid),
			pg_catalog.pg_relation_size(c.oid)
		FROM pg_catalog.pg_class c
			JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('r', 'p')
			AND n.nspname NOT IN (` + SystemSchemaWhereClause + `)
		ORDER BY n.nspname, c.relname`

	rows, err := txn.Query(query)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	tableMap := make(map[string][]*schemaextract.Table)

	for rows.Next() {
		var schemaName, tableName string
		var comment sql.NullString
		var totalSize, dataSize int64

		if err := rows.Scan(&schemaName, &tableName, &comment, &totalSize, &dataSize); err != nil {
			return nil, err
		}

		key := TableKey{Schema: schemaName, Table: tableName}

		table := &schemaextract.Table{
			Name:             tableName,
			Columns:          columnMap[key],
			Indexes:          indexMap[key],
			ForeignKeys:      foreignKeyMap[key],
			CheckConstraints: checkMap[key],
			Triggers:         triggerMap[key],
			Partitions:       partitionMap[key],
			Comment:          comment.String,
			DataSize:         dataSize,
			IndexSize:        totalSize - dataSize,
		}

		tableMap[schemaName] = append(tableMap[schemaName], table)
	}
	if err := rows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}

	return tableMap, nil
}

func (e *Extractor) getViews(txn *sql.Tx, columnMap map[TableKey][]*schemaextract.Column) (map[string][]*schemaextract.View, error) {
	query := `
		SELECT
			n.nspname,
			c.relname,
			pg_catalog.pg_get_viewdef(c.oid),
			pg_catalog.obj_description(c.oid, 'pg_class')
		FROM pg_catalog.pg_class c
			JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'v'
			AND n.nspname NOT IN (` + SystemSchemaWhereClause + `)
		ORDER BY n.nspname, c.relname`

	rows, err := txn.Query(query)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	viewMap := make(map[string][]*schemaextract.View)

	for rows.Next() {
		var schemaName, viewName, definition string
		var comment sql.NullString

		if err := rows.Scan(&schemaName, &viewName, &definition, &comment); err != nil {
			return nil, err
		}

		key := TableKey{Schema: schemaName, Table: viewName}

		view := &schemaextract.View{
			Name:       viewName,
			Definition: definition,
			Comment:    comment.String,
			Columns:    columnMap[key],
		}

		viewMap[schemaName] = append(viewMap[schemaName], view)
	}
	if err := rows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}

	return viewMap, nil
}

func (e *Extractor) getMaterializedViews(txn *sql.Tx, columnMap map[TableKey][]*schemaextract.Column) (map[string][]*schemaextract.MaterializedView, error) {
	query := `
		SELECT
			n.nspname,
			c.relname,
			pg_catalog.pg_get_viewdef(c.oid),
			pg_catalog.obj_description(c.oid, 'pg_class')
		FROM pg_catalog.pg_class c
			JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'm'
			AND n.nspname NOT IN (` + SystemSchemaWhereClause + `)
		ORDER BY n.nspname, c.relname`

	rows, err := txn.Query(query)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	mvMap := make(map[string][]*schemaextract.MaterializedView)

	for rows.Next() {
		var schemaName, mvName, definition string
		var comment sql.NullString

		if err := rows.Scan(&schemaName, &mvName, &definition, &comment); err != nil {
			return nil, err
		}

		key := TableKey{Schema: schemaName, Table: mvName}

		mv := &schemaextract.MaterializedView{
			Name:       mvName,
			Definition: definition,
			Comment:    comment.String,
			Columns:    columnMap[key],
		}

		mvMap[schemaName] = append(mvMap[schemaName], mv)
	}
	if err := rows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}

	return mvMap, nil
}

type extensionDependency struct {
	objectType string
	schemaName string
	objectName string
}

func (e *Extractor) getExtensionDepend(txn *sql.Tx) (map[extensionDependency]bool, error) {
	query := `
		SELECT
			CASE
				WHEN c.relkind = 'r' THEN 'TABLE'
				WHEN c.relkind = 'v' THEN 'VIEW'
				WHEN c.relkind = 'm' THEN 'MATERIALIZED VIEW'
				WHEN p.prokind = 'f' THEN 'FUNCTION'
				WHEN p.prokind = 'p' THEN 'PROCEDURE'
				ELSE 'UNKNOWN'
			END,
			n.nspname,
			COALESCE(c.relname, p.proname)
		FROM pg_catalog.pg_depend d
			LEFT JOIN pg_catalog.pg_class c ON c.oid = d.objid
			LEFT JOIN pg_catalog.pg_proc p ON p.oid = d.objid
			LEFT JOIN pg_catalog.pg_namespace n ON n.oid = COALESCE(c.relnamespace, p.pronamespace)
			JOIN pg_catalog.pg_extension e ON e.oid = d.refobjid
		WHERE d.deptype = 'e'
			AND n.nspname NOT IN (` + SystemSchemaWhereClause + `)`

	rows, err := txn.Query(query)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	extDepend := make(map[extensionDependency]bool)

	for rows.Next() {
		var objectType, schemaName, objectName string
		if err := rows.Scan(&objectType, &schemaName, &objectName); err != nil {
			return nil, err
		}

		dep := extensionDependency{
			objectType: objectType,
			schemaName: schemaName,
			objectName: objectName,
		}
		extDepend[dep] = true
	}
	if err := rows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}

	return extDepend, nil
}

func (e *Extractor) getFunctions(txn *sql.Tx, extensionDepend map[extensionDependency]bool) (map[string][]*schemaextract.Function, error) {
	query := `
		SELECT
			n.nspname,
			p.proname,
			pg_catalog.pg_get_functiondef(p.oid),
			pg_catalog.obj_description(p.oid, 'pg_proc')
		FROM pg_catalog.pg_proc p
			JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
		WHERE p.prokind = 'f'
			AND n.nspname NOT IN (` + SystemSchemaWhereClause + `)
		ORDER BY n.nspname, p.proname`

	rows, err := txn.Query(query)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	functionMap := make(map[string][]*schemaextract.Function)

	for rows.Next() {
		var schemaName, functionName, definition string
		var comment sql.NullString

		if err := rows.Scan(&schemaName, &functionName, &definition, &comment); err != nil {
			return nil, err
		}

		// Skip extension-dependent functions
		dep := extensionDependency{
			objectType: "FUNCTION",
			schemaName: schemaName,
			objectName: functionName,
		}
		if extensionDepend[dep] {
			continue
		}

		// Skip system functions
		if IsSystemFunction(functionName, definition) {
			continue
		}

		function := &schemaextract.Function{
			Name:       functionName,
			Definition: definition,
			Comment:    comment.String,
		}

		functionMap[schemaName] = append(functionMap[schemaName], function)
	}
	if err := rows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}

	return functionMap, nil
}

func (e *Extractor) getProcedures(txn *sql.Tx, extensionDepend map[extensionDependency]bool) (map[string][]*schemaextract.Procedure, error) {
	query := `
		SELECT
			n.nspname,
			p.proname,
			pg_catalog.pg_get_functiondef(p.oid),
			pg_catalog.obj_description(p.oid, 'pg_proc')
		FROM pg_catalog.pg_proc p
			JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
		WHERE p.prokind = 'p'
			AND n.nspname NOT IN (` + SystemSchemaWhereClause + `)
		ORDER BY n.nspname, p.proname`

	rows, err := txn.Query(query)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	procedureMap := make(map[string][]*schemaextract.Procedure)

	for rows.Next() {
		var schemaName, procedureName, definition string
		var comment sql.NullString

		if err := rows.Scan(&schemaName, &procedureName, &definition, &comment); err != nil {
			return nil, err
		}

		// Skip extension-dependent procedures
		dep := extensionDependency{
			objectType: "PROCEDURE",
			schemaName: schemaName,
			objectName: procedureName,
		}
		if extensionDepend[dep] {
			continue
		}

		procedure := &schemaextract.Procedure{
			Name:       procedureName,
			Definition: definition,
			Comment:    comment.String,
		}

		procedureMap[schemaName] = append(procedureMap[schemaName], procedure)
	}
	if err := rows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}

	return procedureMap, nil
}

func (e *Extractor) getSequences(txn *sql.Tx) (map[string][]*schemaextract.Sequence, error) {
	query := `
		SELECT
			n.nspname,
			c.relname,
			format_type(s.seqtypid, NULL),
			s.seqstart,
			s.seqincrement,
			s.seqmin,
			s.seqmax,
			s.seqcache,
			s.seqcycle,
			pg_catalog.obj_description(c.oid, 'pg_class')
		FROM pg_catalog.pg_sequence s
			JOIN pg_catalog.pg_class c ON c.oid = s.seqrelid
			JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname NOT IN (` + SystemSchemaWhereClause + `)
		ORDER BY n.nspname, c.relname`

	rows, err := txn.Query(query)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	sequenceMap := make(map[string][]*schemaextract.Sequence)

	for rows.Next() {
		var schemaName, sequenceName, dataType string
		var start, increment, min, max, cache int64
		var cycle bool
		var comment sql.NullString

		if err := rows.Scan(&schemaName, &sequenceName, &dataType, &start, &increment, &min, &max, &cache, &cycle, &comment); err != nil {
			return nil, err
		}

		sequence := &schemaextract.Sequence{
			Name:      sequenceName,
			DataType:  dataType,
			Start:     start,
			Increment: increment,
			Min:       min,
			Max:       max,
			Cache:     cache,
			Cycle:     cycle,
			Comment:   comment.String,
		}

		sequenceMap[schemaName] = append(sequenceMap[schemaName], sequence)
	}
	if err := rows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}

	return sequenceMap, nil
}

func (e *Extractor) getExtensions(txn *sql.Tx) (map[string][]*schemaextract.Extension, error) {
	query := `
		SELECT
			n.nspname,
			e.extname,
			e.extversion,
			pg_catalog.obj_description(e.oid, 'pg_extension')
		FROM pg_catalog.pg_extension e
			JOIN pg_catalog.pg_namespace n ON n.oid = e.extnamespace
		WHERE n.nspname NOT IN (` + SystemSchemaWhereClause + `)
		ORDER BY n.nspname, e.extname`

	rows, err := txn.Query(query)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	extensionMap := make(map[string][]*schemaextract.Extension)

	for rows.Next() {
		var schemaName, extName, version string
		var comment sql.NullString

		if err := rows.Scan(&schemaName, &extName, &version, &comment); err != nil {
			return nil, err
		}

		extension := &schemaextract.Extension{
			Name:    extName,
			Schema:  schemaName,
			Version: version,
			Comment: comment.String,
		}

		extensionMap[schemaName] = append(extensionMap[schemaName], extension)
	}
	if err := rows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}

	return extensionMap, nil
}

func (e *Extractor) getEnumTypes(txn *sql.Tx) (map[string][]*schemaextract.EnumType, error) {
	query := `
		SELECT
			n.nspname,
			t.typname,
			ARRAY_AGG(e.enumlabel ORDER BY e.enumsortorder)
		FROM pg_catalog.pg_type t
			JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
			JOIN pg_catalog.pg_enum e ON e.enumtypid = t.oid
		WHERE t.typtype = 'e'
			AND n.nspname NOT IN (` + SystemSchemaWhereClause + `)
		GROUP BY n.nspname, t.typname
		ORDER BY n.nspname, t.typname`

	rows, err := txn.Query(query)
	if err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	enumMap := make(map[string][]*schemaextract.EnumType)

	for rows.Next() {
		var schemaName, enumName string
		var values pq.StringArray

		if err := rows.Scan(&schemaName, &enumName, &values); err != nil {
			return nil, err
		}

		enum := &schemaextract.EnumType{
			Name:   enumName,
			Schema: schemaName,
			Values: []string(values),
		}

		enumMap[schemaName] = append(enumMap[schemaName], enum)
	}
	if err := rows.Err(); err != nil {
		return nil, comparer.FormatErrorWithQuery(err, query)
	}

	return enumMap, nil
}
