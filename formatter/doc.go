// Package formatter provides human-readable formatting for schema
// differences and comparison reports.
//
// # Overview
//
// The formatter package takes schema differences and produces formatted
// output in various styles suitable for different use cases:
//   - Text: Plain text for console output
//   - Markdown: For documentation and pull requests
//   - JSON: For programmatic consumption
//   - HTML: For web interfaces (future)
//
// # Usage
//
// Format differences as text:
//
//	diffs, _ := comparer.CompareSchemas(old, new, nil)
//	report := formatter.FormatText(diffs, &formatter.Options{
//	    ShowContext: true,
//	    Colorize:    true,
//	})
//	fmt.Println(report)
//
// Format as markdown:
//
//	mdReport := formatter.FormatMarkdown(diffs, &formatter.Options{
//	    GroupByCategory: true,
//	    IncludeSummary:  true,
//	})
//	os.WriteFile("schema-changes.md", []byte(mdReport), 0644)
//
// Format as JSON:
//
//	jsonReport := formatter.FormatJSON(diffs, &formatter.Options{
//	    PrettyPrint: true,
//	})
//
// # Text Output Features
//
//   - Color-coded severity (red for critical, yellow for warning)
//   - Grouped by category (tables, columns, indexes)
//   - Summary statistics (X tables added, Y columns modified)
//   - Optional context (schema names, object details)
//
// # Markdown Output Features
//
//   - Proper heading hierarchy
//   - Tables for structured data
//   - Code blocks for SQL snippets
//   - Emoji indicators for severity
//   - Collapsible sections for large reports
//
// # Formatting Options
//
// Options control output format:
//   - ShowContext: Include schema and table context
//   - Colorize: Use ANSI color codes (text only)
//   - GroupByCategory: Group by object type vs chronological
//   - IncludeSummary: Add summary section
//   - Verbose: Include all details vs high-level overview
//
// # Use Cases
//
//   - Console output for developers
//   - Pull request descriptions
//   - Migration documentation
//   - Audit reports
//   - Automated notifications (Slack, email)
package formatter
