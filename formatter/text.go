package formatter

import (
	"fmt"
	"strings"

	"github.com/nsxbet/sql-schema/diff"
	"github.com/nsxbet/sql-schema/planner"
)

// FormatDiffAsText formats a MetadataDiff as human-readable text.
func FormatDiffAsText(mdiff *diff.MetadataDiff) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Database Schema Comparison: %s\n", mdiff.DatabaseName))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")

	// Summary
	totalChanges := countTotalChanges(mdiff)
	sb.WriteString(fmt.Sprintf("Total Changes: %d\n\n", totalChanges))

	// Schema changes
	if len(mdiff.SchemaChanges) > 0 {
		sb.WriteString(formatSection("Schema Changes", mdiff.SchemaChanges,
			func(d *diff.SchemaDiff) string {
				return fmt.Sprintf("[%s] Schema: %s", d.Action, d.SchemaName)
			}))
	}

	// Table changes
	if len(mdiff.TableChanges) > 0 {
		sb.WriteString(formatTableChanges(mdiff.TableChanges))
	}

	// View changes
	if len(mdiff.ViewChanges) > 0 {
		sb.WriteString(formatSection("View Changes", mdiff.ViewChanges,
			func(d *diff.ViewDiff) string {
				return fmt.Sprintf("[%s] View: %s.%s", d.Action, d.SchemaName, d.ViewName)
			}))
	}

	// Materialized view changes
	if len(mdiff.MaterializedViewChanges) > 0 {
		sb.WriteString(formatSection("Materialized View Changes", mdiff.MaterializedViewChanges,
			func(d *diff.MaterializedViewDiff) string {
				return fmt.Sprintf("[%s] Materialized View: %s.%s", d.Action, d.SchemaName, d.ViewName)
			}))
	}

	// Function changes
	if len(mdiff.FunctionChanges) > 0 {
		sb.WriteString(formatSection("Function Changes", mdiff.FunctionChanges,
			func(d *diff.FunctionDiff) string {
				name := "unknown"
				if d.NewFunction != nil {
					name = d.NewFunction.Name
				} else if d.OldFunction != nil {
					name = d.OldFunction.Name
				}
				return fmt.Sprintf("[%s] Function: %s", d.Action, name)
			}))
	}

	// Procedure changes
	if len(mdiff.ProcedureChanges) > 0 {
		sb.WriteString(formatSection("Procedure Changes", mdiff.ProcedureChanges,
			func(d *diff.ProcedureDiff) string {
				name := "unknown"
				if d.NewProcedure != nil {
					name = d.NewProcedure.Name
				} else if d.OldProcedure != nil {
					name = d.OldProcedure.Name
				}
				return fmt.Sprintf("[%s] Procedure: %s", d.Action, name)
			}))
	}

	// Sequence changes
	if len(mdiff.SequenceChanges) > 0 {
		sb.WriteString(formatSection("Sequence Changes", mdiff.SequenceChanges,
			func(d *diff.SequenceDiff) string {
				name := "unknown"
				if d.NewSequence != nil {
					name = d.NewSequence.Name
				} else if d.OldSequence != nil {
					name = d.OldSequence.Name
				}
				return fmt.Sprintf("[%s] Sequence: %s", d.Action, name)
			}))
	}

	// Enum type changes
	if len(mdiff.EnumTypeChanges) > 0 {
		sb.WriteString(formatSection("Enum Type Changes", mdiff.EnumTypeChanges,
			func(d *diff.EnumTypeDiff) string {
				return fmt.Sprintf("[%s] Enum: %s.%s", d.Action, d.SchemaName, d.EnumName)
			}))
	}

	// Event changes (MySQL)
	if len(mdiff.EventChanges) > 0 {
		sb.WriteString(formatSection("Event Changes", mdiff.EventChanges,
			func(d *diff.EventDiff) string {
				name := "unknown"
				if d.NewEvent != nil {
					name = d.NewEvent.Name
				} else if d.OldEvent != nil {
					name = d.OldEvent.Name
				}
				return fmt.Sprintf("[%s] Event: %s", d.Action, name)
			}))
	}

	// Extension changes (PostgreSQL)
	if len(mdiff.ExtensionChanges) > 0 {
		sb.WriteString(formatSection("Extension Changes", mdiff.ExtensionChanges,
			func(d *diff.ExtensionDiff) string {
				return fmt.Sprintf("[%s] Extension: %s", d.Action, d.ExtensionName)
			}))
	}

	return sb.String()
}

// formatSection formats a generic section with a title and list of items.
func formatSection[T any](title string, items []T, formatter func(T) string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s (%d)\n", title, len(items)))
	sb.WriteString(strings.Repeat("-", 50) + "\n")
	for _, item := range items {
		sb.WriteString("  " + formatter(item) + "\n")
	}
	sb.WriteString("\n")
	return sb.String()
}

// formatTableChanges formats table changes with detailed sub-changes.
func formatTableChanges(tableChanges []*diff.TableDiff) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Table Changes (%d)\n", len(tableChanges)))
	sb.WriteString(strings.Repeat("-", 50) + "\n")

	for _, tc := range tableChanges {
		sb.WriteString(fmt.Sprintf("  [%s] Table: %s.%s\n", tc.Action, tc.SchemaName, tc.TableName))

		// Column changes
		if len(tc.ColumnChanges) > 0 {
			sb.WriteString(fmt.Sprintf("    Columns (%d):\n", len(tc.ColumnChanges)))
			for _, cc := range tc.ColumnChanges {
				colName := "unknown"
				if cc.NewColumn != nil {
					colName = cc.NewColumn.Name
				} else if cc.OldColumn != nil {
					colName = cc.OldColumn.Name
				}
				sb.WriteString(fmt.Sprintf("      [%s] %s\n", cc.Action, colName))
			}
		}

		// Index changes
		if len(tc.IndexChanges) > 0 {
			sb.WriteString(fmt.Sprintf("    Indexes (%d):\n", len(tc.IndexChanges)))
			for _, ic := range tc.IndexChanges {
				idxName := "unknown"
				if ic.NewIndex != nil {
					idxName = ic.NewIndex.Name
				} else if ic.OldIndex != nil {
					idxName = ic.OldIndex.Name
				}
				sb.WriteString(fmt.Sprintf("      [%s] %s\n", ic.Action, idxName))
			}
		}

		// Foreign key changes
		if len(tc.ForeignKeyChanges) > 0 {
			sb.WriteString(fmt.Sprintf("    Foreign Keys (%d):\n", len(tc.ForeignKeyChanges)))
			for _, fk := range tc.ForeignKeyChanges {
				fkName := "unknown"
				if fk.NewForeignKey != nil {
					fkName = fk.NewForeignKey.Name
				} else if fk.OldForeignKey != nil {
					fkName = fk.OldForeignKey.Name
				}
				sb.WriteString(fmt.Sprintf("      [%s] %s\n", fk.Action, fkName))
			}
		}

		// Check constraint changes
		if len(tc.CheckConstraintChanges) > 0 {
			sb.WriteString(fmt.Sprintf("    Check Constraints (%d):\n", len(tc.CheckConstraintChanges)))
			for _, chk := range tc.CheckConstraintChanges {
				chkName := "unknown"
				if chk.NewConstraint != nil {
					chkName = chk.NewConstraint.Name
				} else if chk.OldConstraint != nil {
					chkName = chk.OldConstraint.Name
				}
				sb.WriteString(fmt.Sprintf("      [%s] %s\n", chk.Action, chkName))
			}
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatStrategyAsText formats a MigrationStrategy as human-readable text.
func FormatStrategyAsText(strategy *planner.MigrationStrategy) string {
	var sb strings.Builder

	sb.WriteString("Migration Strategy Report\n")
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")

	sb.WriteString(fmt.Sprintf("Engine: %s\n", strategy.Engine))
	sb.WriteString(fmt.Sprintf("Total Operations: %d\n", len(strategy.Operations)))
	sb.WriteString(fmt.Sprintf("High-Risk Operations: %d\n", len(strategy.HighRiskOps)))
	sb.WriteString(fmt.Sprintf("Data Loss Operations: %d\n", len(strategy.DataLossOps)))
	sb.WriteString("\n")

	// Warnings
	if len(strategy.Warnings) > 0 {
		sb.WriteString("⚠️  Global Warnings:\n")
		for _, warning := range strategy.Warnings {
			sb.WriteString(fmt.Sprintf("  - %s\n", warning))
		}
		sb.WriteString("\n")
	}

	// High-risk operations
	if len(strategy.HighRiskOps) > 0 {
		sb.WriteString("🔴 High-Risk Operations:\n")
		sb.WriteString(strings.Repeat("-", 50) + "\n")
		for i, op := range strategy.HighRiskOps {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, op.Description))
			sb.WriteString(fmt.Sprintf("   Risk: %s - %s\n", op.Risk, op.RiskReason))
			if len(op.Warnings) > 0 {
				for _, warning := range op.Warnings {
					sb.WriteString(fmt.Sprintf("   ⚠️  %s\n", warning))
				}
			}
			sb.WriteString("\n")
		}
	}

	// Execution order
	sb.WriteString("📋 Execution Order:\n")
	sb.WriteString(strings.Repeat("-", 50) + "\n")
	for i, op := range strategy.OrderedOperations {
		riskIcon := getRiskIcon(op.Risk)
		reversibleIcon := "✓"
		if !op.Reversible {
			reversibleIcon = "✗"
		}
		sb.WriteString(fmt.Sprintf("%d. [%s] %s %s (Reversible: %s)\n",
			i+1, op.Action, riskIcon, op.Description, reversibleIcon))

		if len(op.Dependencies) > 0 {
			sb.WriteString(fmt.Sprintf("   Dependencies: %v\n", op.Dependencies))
		}
	}
	sb.WriteString("\n")

	return sb.String()
}

// getRiskIcon returns an icon for the risk level.
func getRiskIcon(risk planner.RiskLevel) string {
	switch risk {
	case planner.RiskLevelHigh:
		return "🔴"
	case planner.RiskLevelMedium:
		return "🟡"
	case planner.RiskLevelLow:
		return "🟢"
	case planner.RiskLevelNone:
		return "⚪"
	default:
		return "⚫"
	}
}

// countTotalChanges counts all changes in a MetadataDiff.
func countTotalChanges(mdiff *diff.MetadataDiff) int {
	total := len(mdiff.SchemaChanges) +
		len(mdiff.TableChanges) +
		len(mdiff.ViewChanges) +
		len(mdiff.MaterializedViewChanges) +
		len(mdiff.FunctionChanges) +
		len(mdiff.ProcedureChanges) +
		len(mdiff.SequenceChanges) +
		len(mdiff.EnumTypeChanges) +
		len(mdiff.EventChanges) +
		len(mdiff.ExtensionChanges) +
		len(mdiff.CommentChanges)

	// Add sub-changes from tables
	for _, tc := range mdiff.TableChanges {
		total += len(tc.ColumnChanges) +
			len(tc.IndexChanges) +
			len(tc.ForeignKeyChanges) +
			len(tc.CheckConstraintChanges) +
			len(tc.PrimaryKeyChanges) +
			len(tc.UniqueConstraintChanges)
	}

	return total
}
