package util

import (
	"fmt"
	"regexp"
	"strings"

	schemaextract "github.com/nsxbet/sql-schema"
)

// FilterOptions configures schema filtering.
type FilterOptions struct {
	// IncludeSchemas includes only these schemas (empty = all)
	IncludeSchemas []string
	// ExcludeSchemas excludes these schemas
	ExcludeSchemas []string

	// IncludeTables includes only tables matching these patterns
	IncludeTables []*regexp.Regexp
	// ExcludeTables excludes tables matching these patterns
	ExcludeTables []*regexp.Regexp

	// IncludeViews includes only views matching these patterns
	IncludeViews []*regexp.Regexp
	// ExcludeViews excludes views matching these patterns
	ExcludeViews []*regexp.Regexp

	// IncludeFunctions includes only functions matching these patterns
	IncludeFunctions []*regexp.Regexp
	// ExcludeFunctions excludes functions matching these patterns
	ExcludeFunctions []*regexp.Regexp

	// TableMinSize filters tables smaller than this size in bytes (0 = no filter)
	TableMinSize int64
	// TableMaxSize filters tables larger than this size in bytes (0 = no filter)
	TableMaxSize int64

	// RemoveEmptySchemas removes schemas with no objects after filtering
	RemoveEmptySchemas bool
}

// FilterSchema applies filtering options to a database schema and returns a new filtered schema.
func FilterSchema(schema *schemaextract.DatabaseSchema, opts *FilterOptions) *schemaextract.DatabaseSchema {
	if opts == nil {
		return schema
	}

	filtered := &schemaextract.DatabaseSchema{
		Name:         schema.Name,
		CharacterSet: schema.CharacterSet,
		Collation:    schema.Collation,
		SearchPath:   schema.SearchPath,
		Schemas:      make([]*schemaextract.Schema, 0),
	}

	for _, s := range schema.Schemas {
		if !shouldIncludeSchema(s.Name, opts) {
			continue
		}

		filteredSchema := filterSchemaObjects(s, opts)

		// Skip empty schemas if configured
		if opts.RemoveEmptySchemas && isSchemaEmpty(filteredSchema) {
			continue
		}

		filtered.Schemas = append(filtered.Schemas, filteredSchema)
	}

	return filtered
}

func shouldIncludeSchema(schemaName string, opts *FilterOptions) bool {
	// Check exclude list first
	for _, exclude := range opts.ExcludeSchemas {
		if schemaName == exclude {
			return false
		}
	}

	// If include list is empty, include all (except excluded)
	if len(opts.IncludeSchemas) == 0 {
		return true
	}

	// Check include list
	for _, include := range opts.IncludeSchemas {
		if schemaName == include {
			return true
		}
	}

	return false
}

func filterSchemaObjects(schema *schemaextract.Schema, opts *FilterOptions) *schemaextract.Schema {
	filtered := &schemaextract.Schema{
		Name:              schema.Name,
		Tables:            filterTables(schema.Tables, opts),
		Views:             filterViews(schema.Views, opts),
		MaterializedViews: filterMaterializedViews(schema.MaterializedViews, opts),
		Functions:         filterFunctions(schema.Functions, opts),
		Procedures:        schema.Procedures, // Keep all procedures for now
		Sequences:         schema.Sequences,  // Keep all sequences
		Extensions:        schema.Extensions, // Keep all extensions
		EnumTypes:         schema.EnumTypes,  // Keep all enum types
		Events:            schema.Events,     // Keep all events
	}

	return filtered
}

func filterTables(tables []*schemaextract.Table, opts *FilterOptions) []*schemaextract.Table {
	if len(opts.IncludeTables) == 0 && len(opts.ExcludeTables) == 0 && opts.TableMinSize == 0 && opts.TableMaxSize == 0 {
		return tables
	}

	filtered := make([]*schemaextract.Table, 0)

	for _, table := range tables {
		// Check exclude patterns first
		if matchesAnyPattern(table.Name, opts.ExcludeTables) {
			continue
		}

		// Check include patterns (if specified)
		if len(opts.IncludeTables) > 0 && !matchesAnyPattern(table.Name, opts.IncludeTables) {
			continue
		}

		// Check size constraints
		if opts.TableMinSize > 0 && table.DataSize < opts.TableMinSize {
			continue
		}
		if opts.TableMaxSize > 0 && table.DataSize > opts.TableMaxSize {
			continue
		}

		filtered = append(filtered, table)
	}

	return filtered
}

func filterViews(views []*schemaextract.View, opts *FilterOptions) []*schemaextract.View {
	if len(opts.IncludeViews) == 0 && len(opts.ExcludeViews) == 0 {
		return views
	}

	filtered := make([]*schemaextract.View, 0)

	for _, view := range views {
		// Check exclude patterns first
		if matchesAnyPattern(view.Name, opts.ExcludeViews) {
			continue
		}

		// Check include patterns (if specified)
		if len(opts.IncludeViews) > 0 && !matchesAnyPattern(view.Name, opts.IncludeViews) {
			continue
		}

		filtered = append(filtered, view)
	}

	return filtered
}

func filterMaterializedViews(views []*schemaextract.MaterializedView, opts *FilterOptions) []*schemaextract.MaterializedView {
	if len(opts.IncludeViews) == 0 && len(opts.ExcludeViews) == 0 {
		return views
	}

	filtered := make([]*schemaextract.MaterializedView, 0)

	for _, view := range views {
		// Check exclude patterns first
		if matchesAnyPattern(view.Name, opts.ExcludeViews) {
			continue
		}

		// Check include patterns (if specified)
		if len(opts.IncludeViews) > 0 && !matchesAnyPattern(view.Name, opts.IncludeViews) {
			continue
		}

		filtered = append(filtered, view)
	}

	return filtered
}

func filterFunctions(functions []*schemaextract.Function, opts *FilterOptions) []*schemaextract.Function {
	if len(opts.IncludeFunctions) == 0 && len(opts.ExcludeFunctions) == 0 {
		return functions
	}

	filtered := make([]*schemaextract.Function, 0)

	for _, fn := range functions {
		// Check exclude patterns first
		if matchesAnyPattern(fn.Name, opts.ExcludeFunctions) {
			continue
		}

		// Check include patterns (if specified)
		if len(opts.IncludeFunctions) > 0 && !matchesAnyPattern(fn.Name, opts.IncludeFunctions) {
			continue
		}

		filtered = append(filtered, fn)
	}

	return filtered
}

func matchesAnyPattern(name string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

func isSchemaEmpty(schema *schemaextract.Schema) bool {
	return len(schema.Tables) == 0 &&
		len(schema.Views) == 0 &&
		len(schema.MaterializedViews) == 0 &&
		len(schema.Functions) == 0 &&
		len(schema.Procedures) == 0 &&
		len(schema.Sequences) == 0 &&
		len(schema.Extensions) == 0 &&
		len(schema.EnumTypes) == 0 &&
		len(schema.Events) == 0
}

// NewFilterOptions creates a new FilterOptions with default values.
func NewFilterOptions() *FilterOptions {
	return &FilterOptions{
		RemoveEmptySchemas: true,
	}
}

// WithIncludeSchemas sets the schemas to include.
func (opts *FilterOptions) WithIncludeSchemas(schemas ...string) *FilterOptions {
	opts.IncludeSchemas = schemas
	return opts
}

// WithExcludeSchemas sets the schemas to exclude.
func (opts *FilterOptions) WithExcludeSchemas(schemas ...string) *FilterOptions {
	opts.ExcludeSchemas = schemas
	return opts
}

// WithIncludeTablesPattern adds a table name pattern to include.
func (opts *FilterOptions) WithIncludeTablesPattern(pattern string) *FilterOptions {
	re, err := regexp.Compile(pattern)
	if err == nil {
		opts.IncludeTables = append(opts.IncludeTables, re)
	}
	return opts
}

// WithExcludeTablesPattern adds a table name pattern to exclude.
func (opts *FilterOptions) WithExcludeTablesPattern(pattern string) *FilterOptions {
	re, err := regexp.Compile(pattern)
	if err == nil {
		opts.ExcludeTables = append(opts.ExcludeTables, re)
	}
	return opts
}

// WithTableSizeRange sets the minimum and maximum table sizes in bytes.
func (opts *FilterOptions) WithTableSizeRange(minSize, maxSize int64) *FilterOptions {
	opts.TableMinSize = minSize
	opts.TableMaxSize = maxSize
	return opts
}

// WithRemoveEmptySchemas sets whether to remove empty schemas after filtering.
func (opts *FilterOptions) WithRemoveEmptySchemas(remove bool) *FilterOptions {
	opts.RemoveEmptySchemas = remove
	return opts
}

// FilterByPrefix creates a filter that includes only objects with the given prefix.
func FilterByPrefix(prefix string) *FilterOptions {
	pattern := "^" + regexp.QuoteMeta(prefix)
	opts := NewFilterOptions()
	opts.WithIncludeTablesPattern(pattern)
	return opts
}

// FilterBySuffix creates a filter that includes only objects with the given suffix.
func FilterBySuffix(suffix string) *FilterOptions {
	pattern := regexp.QuoteMeta(suffix) + "$"
	opts := NewFilterOptions()
	opts.WithIncludeTablesPattern(pattern)
	return opts
}

// FilterBySize creates a filter that includes only tables within the specified size range.
func FilterBySize(minBytes, maxBytes int64) *FilterOptions {
	opts := NewFilterOptions()
	opts.WithTableSizeRange(minBytes, maxBytes)
	return opts
}

// FilterBySchema creates a filter that includes only the specified schemas.
func FilterBySchema(schemas ...string) *FilterOptions {
	opts := NewFilterOptions()
	opts.WithIncludeSchemas(schemas...)
	return opts
}

// ExcludeSystemSchemas creates a filter that excludes common system schemas.
// For PostgreSQL: pg_catalog, information_schema, pg_toast
// For MySQL: mysql, information_schema, performance_schema, sys
func ExcludeSystemSchemas() *FilterOptions {
	opts := NewFilterOptions()
	opts.WithExcludeSchemas(
		"pg_catalog",
		"information_schema",
		"pg_toast",
		"mysql",
		"performance_schema",
		"sys",
	)
	return opts
}

// ExcludeTestTables creates a filter that excludes tables commonly used for testing.
func ExcludeTestTables() *FilterOptions {
	opts := NewFilterOptions()
	opts.WithExcludeTablesPattern("^test_")
	opts.WithExcludeTablesPattern("^tmp_")
	opts.WithExcludeTablesPattern("_test$")
	opts.WithExcludeTablesPattern("_tmp$")
	return opts
}

// CombineFilters combines multiple filter options into one.
// Later filters take precedence over earlier ones for conflicting settings.
func CombineFilters(filters ...*FilterOptions) *FilterOptions {
	if len(filters) == 0 {
		return NewFilterOptions()
	}

	combined := NewFilterOptions()

	for _, filter := range filters {
		if filter == nil {
			continue
		}

		// Merge include/exclude lists
		combined.IncludeSchemas = append(combined.IncludeSchemas, filter.IncludeSchemas...)
		combined.ExcludeSchemas = append(combined.ExcludeSchemas, filter.ExcludeSchemas...)
		combined.IncludeTables = append(combined.IncludeTables, filter.IncludeTables...)
		combined.ExcludeTables = append(combined.ExcludeTables, filter.ExcludeTables...)
		combined.IncludeViews = append(combined.IncludeViews, filter.IncludeViews...)
		combined.ExcludeViews = append(combined.ExcludeViews, filter.ExcludeViews...)
		combined.IncludeFunctions = append(combined.IncludeFunctions, filter.IncludeFunctions...)
		combined.ExcludeFunctions = append(combined.ExcludeFunctions, filter.ExcludeFunctions...)

		// Take the most restrictive size constraints
		if filter.TableMinSize > combined.TableMinSize {
			combined.TableMinSize = filter.TableMinSize
		}
		if filter.TableMaxSize > 0 && (combined.TableMaxSize == 0 || filter.TableMaxSize < combined.TableMaxSize) {
			combined.TableMaxSize = filter.TableMaxSize
		}

		// Last filter wins for boolean flags
		combined.RemoveEmptySchemas = filter.RemoveEmptySchemas
	}

	// Remove duplicates from string slices
	combined.IncludeSchemas = uniqueStrings(combined.IncludeSchemas)
	combined.ExcludeSchemas = uniqueStrings(combined.ExcludeSchemas)

	return combined
}

func uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// Summary returns a human-readable summary of the filter configuration.
func (opts *FilterOptions) Summary() string {
	if opts == nil {
		return "No filters applied"
	}

	var parts []string

	if len(opts.IncludeSchemas) > 0 {
		parts = append(parts, "Include schemas: "+strings.Join(opts.IncludeSchemas, ", "))
	}
	if len(opts.ExcludeSchemas) > 0 {
		parts = append(parts, "Exclude schemas: "+strings.Join(opts.ExcludeSchemas, ", "))
	}
	if len(opts.IncludeTables) > 0 {
		parts = append(parts, fmt.Sprintf("Include tables: %d patterns", len(opts.IncludeTables)))
	}
	if len(opts.ExcludeTables) > 0 {
		parts = append(parts, fmt.Sprintf("Exclude tables: %d patterns", len(opts.ExcludeTables)))
	}
	if opts.TableMinSize > 0 {
		parts = append(parts, fmt.Sprintf("Min table size: %d bytes", opts.TableMinSize))
	}
	if opts.TableMaxSize > 0 {
		parts = append(parts, fmt.Sprintf("Max table size: %d bytes", opts.TableMaxSize))
	}

	if len(parts) == 0 {
		return "No filters applied"
	}

	return strings.Join(parts, "; ")
}
