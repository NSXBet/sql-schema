package formatter

import (
	"fmt"
	"strings"

	"github.com/nsxbet/sql-schema/diff"
	"github.com/nsxbet/sql-schema/planner"
)

// FormatDiffAsMarkdown formats a MetadataDiff as Markdown.
func FormatDiffAsMarkdown(mdiff *diff.MetadataDiff) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Database Schema Comparison: %s\n\n", mdiff.DatabaseName))

	// Summary
	totalChanges := countTotalChanges(mdiff)
	sb.WriteString("## Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Total Changes:** %d\n", totalChanges))
	sb.WriteString(fmt.Sprintf("- **Schemas:** %d changes\n", len(mdiff.SchemaChanges)))
	sb.WriteString(fmt.Sprintf("- **Tables:** %d changes\n", len(mdiff.TableChanges)))
	sb.WriteString(fmt.Sprintf("- **Views:** %d changes\n", len(mdiff.ViewChanges)))
	sb.WriteString(fmt.Sprintf("- **Functions:** %d changes\n", len(mdiff.FunctionChanges)))
	sb.WriteString(fmt.Sprintf("- **Procedures:** %d changes\n", len(mdiff.ProcedureChanges)))
	sb.WriteString("\n")

	// Schema changes
	if len(mdiff.SchemaChanges) > 0 {
		sb.WriteString("## Schema Changes\n\n")
		sb.WriteString("| Action | Schema Name |\n")
		sb.WriteString("|--------|-------------|\n")
		for _, sc := range mdiff.SchemaChanges {
			action := formatActionBadge(sc.Action)
			sb.WriteString(fmt.Sprintf("| %s | `%s` |\n", action, sc.SchemaName))
		}
		sb.WriteString("\n")
	}

	// Table changes
	if len(mdiff.TableChanges) > 0 {
		sb.WriteString("## Table Changes\n\n")
		for _, tc := range mdiff.TableChanges {
			action := formatActionBadge(tc.Action)
			sb.WriteString(fmt.Sprintf("### %s Table: `%s.%s`\n\n", action, tc.SchemaName, tc.TableName))

			// Column changes
			if len(tc.ColumnChanges) > 0 {
				sb.WriteString("#### Column Changes\n\n")
				sb.WriteString("| Action | Column | Type | Nullable | Default |\n")
				sb.WriteString("|--------|--------|------|----------|----------|\n")
				for _, cc := range tc.ColumnChanges {
					colName, colType, nullable, defaultVal := getColumnDetails(cc)
					actionBadge := formatActionBadge(cc.Action)
					sb.WriteString(fmt.Sprintf("| %s | `%s` | `%s` | %s | `%s` |\n",
						actionBadge, colName, colType, nullable, defaultVal))
				}
				sb.WriteString("\n")
			}

			// Index changes
			if len(tc.IndexChanges) > 0 {
				sb.WriteString("#### Index Changes\n\n")
				sb.WriteString("| Action | Index Name | Type | Unique |\n")
				sb.WriteString("|--------|------------|------|--------|\n")
				for _, ic := range tc.IndexChanges {
					idxName, idxType, unique := getIndexDetails(ic)
					actionBadge := formatActionBadge(ic.Action)
					sb.WriteString(fmt.Sprintf("| %s | `%s` | %s | %s |\n",
						actionBadge, idxName, idxType, unique))
				}
				sb.WriteString("\n")
			}

			// Foreign key changes
			if len(tc.ForeignKeyChanges) > 0 {
				sb.WriteString("#### Foreign Key Changes\n\n")
				sb.WriteString("| Action | Name | Referenced Table |\n")
				sb.WriteString("|--------|------|------------------|\n")
				for _, fk := range tc.ForeignKeyChanges {
					fkName, refTable := getForeignKeyDetails(fk)
					actionBadge := formatActionBadge(fk.Action)
					sb.WriteString(fmt.Sprintf("| %s | `%s` | `%s` |\n",
						actionBadge, fkName, refTable))
				}
				sb.WriteString("\n")
			}
		}
	}

	// View changes
	if len(mdiff.ViewChanges) > 0 {
		sb.WriteString("## View Changes\n\n")
		sb.WriteString("| Action | Schema | View Name | Definition Changed |\n")
		sb.WriteString("|--------|--------|-----------|-------------------|\n")
		for _, vc := range mdiff.ViewChanges {
			action := formatActionBadge(vc.Action)
			defChanged := formatBool(vc.DefinitionChanged)
			sb.WriteString(fmt.Sprintf("| %s | `%s` | `%s` | %s |\n",
				action, vc.SchemaName, vc.ViewName, defChanged))
		}
		sb.WriteString("\n")
	}

	// Function changes
	if len(mdiff.FunctionChanges) > 0 {
		sb.WriteString("## Function Changes\n\n")
		sb.WriteString("| Action | Function Name | Body Changed | Can Use ALTER |\n")
		sb.WriteString("|--------|---------------|--------------|---------------|\n")
		for _, fc := range mdiff.FunctionChanges {
			funcName := "unknown"
			if fc.NewFunction != nil {
				funcName = fc.NewFunction.Name
			} else if fc.OldFunction != nil {
				funcName = fc.OldFunction.Name
			}
			action := formatActionBadge(fc.Action)
			bodyChanged := formatBool(fc.BodyChanged)
			canAlter := formatBool(fc.CanUseAlterFunction)
			sb.WriteString(fmt.Sprintf("| %s | `%s` | %s | %s |\n",
				action, funcName, bodyChanged, canAlter))
		}
		sb.WriteString("\n")
	}

	// Procedure changes
	if len(mdiff.ProcedureChanges) > 0 {
		sb.WriteString("## Procedure Changes\n\n")
		sb.WriteString("| Action | Procedure Name | Body Changed | Can Use ALTER |\n")
		sb.WriteString("|--------|----------------|--------------|---------------|\n")
		for _, pc := range mdiff.ProcedureChanges {
			procName := "unknown"
			if pc.NewProcedure != nil {
				procName = pc.NewProcedure.Name
			} else if pc.OldProcedure != nil {
				procName = pc.OldProcedure.Name
			}
			action := formatActionBadge(pc.Action)
			bodyChanged := formatBool(pc.BodyChanged)
			canAlter := formatBool(pc.CanUseAlterProcedure)
			sb.WriteString(fmt.Sprintf("| %s | `%s` | %s | %s |\n",
				action, procName, bodyChanged, canAlter))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatStrategyAsMarkdown formats a MigrationStrategy as Markdown.
func FormatStrategyAsMarkdown(strategy *planner.MigrationStrategy) string {
	var sb strings.Builder

	sb.WriteString("# Migration Strategy Report\n\n")

	// Summary
	sb.WriteString("## Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Engine:** %s\n", strategy.Engine))
	sb.WriteString(fmt.Sprintf("- **Total Operations:** %d\n", len(strategy.Operations)))
	sb.WriteString(fmt.Sprintf("- **High-Risk Operations:** %d\n", len(strategy.HighRiskOps)))
	sb.WriteString(fmt.Sprintf("- **Data Loss Operations:** %d\n", len(strategy.DataLossOps)))
	sb.WriteString("\n")

	// Warnings
	if len(strategy.Warnings) > 0 {
		sb.WriteString("## ⚠️ Warnings\n\n")
		for _, warning := range strategy.Warnings {
			sb.WriteString(fmt.Sprintf("> **Warning:** %s\n\n", warning))
		}
	}

	// High-risk operations
	if len(strategy.HighRiskOps) > 0 {
		sb.WriteString("## 🔴 High-Risk Operations\n\n")
		sb.WriteString("The following operations require special attention:\n\n")
		sb.WriteString("| # | Description | Risk | Reversible | Reason |\n")
		sb.WriteString("|---|-------------|------|------------|--------|\n")
		for i, op := range strategy.HighRiskOps {
			reversible := "❌"
			if op.Reversible {
				reversible = "✅"
			}
			sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s |\n",
				i+1, op.Description, op.Risk, reversible, op.RiskReason))
		}
		sb.WriteString("\n")

		// Show warnings for high-risk ops
		for i, op := range strategy.HighRiskOps {
			if len(op.Warnings) > 0 {
				sb.WriteString(fmt.Sprintf("### Operation %d Warnings\n\n", i+1))
				for _, warning := range op.Warnings {
					sb.WriteString(fmt.Sprintf("- %s\n", warning))
				}
				sb.WriteString("\n")
			}
		}
	}

	// Execution order
	sb.WriteString("## 📋 Execution Order\n\n")
	sb.WriteString("Operations should be executed in the following order:\n\n")
	sb.WriteString("| # | Action | Risk | Description | Reversible | Dependencies |\n")
	sb.WriteString("|---|--------|------|-------------|------------|-------------|\n")
	for i, op := range strategy.OrderedOperations {
		reversible := "✅"
		if !op.Reversible {
			reversible = "❌"
		}
		deps := "-"
		if len(op.Dependencies) > 0 {
			deps = strings.Join(op.Dependencies, ", ")
		}
		riskBadge := formatRiskBadge(op.Risk)
		actionBadge := formatActionBadge(op.Action)
		sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %s |\n",
			i+1, actionBadge, riskBadge, op.Description, reversible, deps))
	}
	sb.WriteString("\n")

	return sb.String()
}

// formatActionBadge returns a Markdown badge for an action.
func formatActionBadge(action diff.MetadataDiffAction) string {
	switch action {
	case diff.MetadataDiffActionCreate:
		return "🟢 `CREATE`"
	case diff.MetadataDiffActionDrop:
		return "🔴 `DROP`"
	case diff.MetadataDiffActionAlter:
		return "🟡 `ALTER`"
	default:
		return "`" + string(action) + "`"
	}
}

// formatRiskBadge returns a Markdown badge for a risk level.
func formatRiskBadge(risk planner.RiskLevel) string {
	switch risk {
	case planner.RiskLevelHigh:
		return "🔴 HIGH"
	case planner.RiskLevelMedium:
		return "🟡 MEDIUM"
	case planner.RiskLevelLow:
		return "🟢 LOW"
	case planner.RiskLevelNone:
		return "⚪ NONE"
	default:
		return string(risk)
	}
}

// formatBool formats a boolean as Yes/No with emoji.
func formatBool(b bool) string {
	if b {
		return "✅ Yes"
	}
	return "❌ No"
}

// getColumnDetails extracts column details from a ColumnDiff.
func getColumnDetails(cc *diff.ColumnDiff) (name, colType, nullable, defaultVal string) {
	name = "unknown"
	colType = ""
	nullable = "?"
	defaultVal = ""

	col := cc.NewColumn
	if col == nil {
		col = cc.OldColumn
	}

	if col != nil {
		name = col.Name
		colType = col.Type
		if col.Nullable {
			nullable = "✅"
		} else {
			nullable = "❌"
		}
		defaultVal = col.Default
	}

	return
}

// getIndexDetails extracts index details from an IndexDiff.
func getIndexDetails(ic *diff.IndexDiff) (name, idxType, unique string) {
	name = "unknown"
	idxType = ""
	unique = "❌"

	idx := ic.NewIndex
	if idx == nil {
		idx = ic.OldIndex
	}

	if idx != nil {
		name = idx.Name
		idxType = idx.Type
		if idx.Unique {
			unique = "✅"
		}
	}

	return
}

// getForeignKeyDetails extracts foreign key details from a ForeignKeyDiff.
func getForeignKeyDetails(fk *diff.ForeignKeyDiff) (name, refTable string) {
	name = "unknown"
	refTable = ""

	fkObj := fk.NewForeignKey
	if fkObj == nil {
		fkObj = fk.OldForeignKey
	}

	if fkObj != nil {
		name = fkObj.Name
		refTable = fkObj.ReferencedTable
	}

	return
}
