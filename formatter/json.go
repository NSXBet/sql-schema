package formatter

import (
	"encoding/json"

	"github.com/nsxbet/sql-schema/diff"
	"github.com/nsxbet/sql-schema/planner"
)

// FormatDiffAsJSON formats a MetadataDiff as JSON.
func FormatDiffAsJSON(mdiff *diff.MetadataDiff, pretty bool) (string, error) {
	var data []byte
	var err error

	if pretty {
		data, err = json.MarshalIndent(mdiff, "", "  ")
	} else {
		data, err = json.Marshal(mdiff)
	}

	if err != nil {
		return "", err
	}

	return string(data), nil
}

// FormatStrategyAsJSON formats a MigrationStrategy as JSON.
func FormatStrategyAsJSON(strategy *planner.MigrationStrategy, pretty bool) (string, error) {
	var data []byte
	var err error

	if pretty {
		data, err = json.MarshalIndent(strategy, "", "  ")
	} else {
		data, err = json.Marshal(strategy)
	}

	if err != nil {
		return "", err
	}

	return string(data), nil
}

// DiffSummary is a simplified view of a MetadataDiff for JSON export.
type DiffSummary struct {
	Database     string                 `json:"database"`
	TotalChanges int                    `json:"totalChanges"`
	Summary      map[string]int         `json:"summary"`
	Changes      map[string]interface{} `json:"changes"`
}

// FormatDiffAsSummaryJSON formats a MetadataDiff as a simplified JSON summary.
func FormatDiffAsSummaryJSON(mdiff *diff.MetadataDiff, pretty bool) (string, error) {
	summary := &DiffSummary{
		Database:     mdiff.DatabaseName,
		TotalChanges: countTotalChanges(mdiff),
		Summary: map[string]int{
			"schemas":           len(mdiff.SchemaChanges),
			"tables":            len(mdiff.TableChanges),
			"views":             len(mdiff.ViewChanges),
			"materializedViews": len(mdiff.MaterializedViewChanges),
			"functions":         len(mdiff.FunctionChanges),
			"procedures":        len(mdiff.ProcedureChanges),
			"sequences":         len(mdiff.SequenceChanges),
			"enumTypes":         len(mdiff.EnumTypeChanges),
			"events":            len(mdiff.EventChanges),
			"extensions":        len(mdiff.ExtensionChanges),
		},
		Changes: make(map[string]interface{}),
	}

	// Add change details
	if len(mdiff.SchemaChanges) > 0 {
		summary.Changes["schemas"] = simplifySchemaChanges(mdiff.SchemaChanges)
	}
	if len(mdiff.TableChanges) > 0 {
		summary.Changes["tables"] = simplifyTableChanges(mdiff.TableChanges)
	}
	if len(mdiff.ViewChanges) > 0 {
		summary.Changes["views"] = simplifyViewChanges(mdiff.ViewChanges)
	}

	var data []byte
	var err error

	if pretty {
		data, err = json.MarshalIndent(summary, "", "  ")
	} else {
		data, err = json.Marshal(summary)
	}

	if err != nil {
		return "", err
	}

	return string(data), nil
}

// StrategySummary is a simplified view of a MigrationStrategy for JSON export.
type StrategySummary struct {
	Engine             string             `json:"engine"`
	TotalOperations    int                `json:"totalOperations"`
	HighRiskCount      int                `json:"highRiskCount"`
	DataLossCount      int                `json:"dataLossCount"`
	RiskDistribution   map[string]int     `json:"riskDistribution"`
	OperationTypes     map[string]int     `json:"operationTypes"`
	Warnings           []string           `json:"warnings,omitempty"`
	HighRiskOperations []OperationSummary `json:"highRiskOperations,omitempty"`
	ExecutionOrder     []OperationSummary `json:"executionOrder"`
}

// OperationSummary is a simplified operation for JSON export.
type OperationSummary struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	Action       string   `json:"action"`
	Description  string   `json:"description"`
	Risk         string   `json:"risk"`
	Reversible   bool     `json:"reversible"`
	Warnings     []string `json:"warnings,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// FormatStrategyAsSummaryJSON formats a MigrationStrategy as a simplified JSON summary.
func FormatStrategyAsSummaryJSON(strategy *planner.MigrationStrategy, pretty bool) (string, error) {
	summary := &StrategySummary{
		Engine:           string(strategy.Engine),
		TotalOperations:  len(strategy.Operations),
		HighRiskCount:    len(strategy.HighRiskOps),
		DataLossCount:    len(strategy.DataLossOps),
		RiskDistribution: make(map[string]int),
		OperationTypes:   make(map[string]int),
		Warnings:         strategy.Warnings,
		ExecutionOrder:   make([]OperationSummary, 0, len(strategy.OrderedOperations)),
	}

	// Count risk distribution and operation types
	for _, op := range strategy.Operations {
		summary.RiskDistribution[string(op.Risk)]++
		summary.OperationTypes[string(op.Type)]++
	}

	// Add high-risk operations
	for _, op := range strategy.HighRiskOps {
		summary.HighRiskOperations = append(summary.HighRiskOperations, toOperationSummary(op))
	}

	// Add execution order
	for _, op := range strategy.OrderedOperations {
		summary.ExecutionOrder = append(summary.ExecutionOrder, toOperationSummary(op))
	}

	var data []byte
	var err error

	if pretty {
		data, err = json.MarshalIndent(summary, "", "  ")
	} else {
		data, err = json.Marshal(summary)
	}

	if err != nil {
		return "", err
	}

	return string(data), nil
}

// toOperationSummary converts a MigrationOperation to OperationSummary.
func toOperationSummary(op *planner.MigrationOperation) OperationSummary {
	return OperationSummary{
		ID:           op.ID,
		Type:         string(op.Type),
		Action:       string(op.Action),
		Description:  op.Description,
		Risk:         string(op.Risk),
		Reversible:   op.Reversible,
		Warnings:     op.Warnings,
		Dependencies: op.Dependencies,
	}
}

// Simplification helpers
type simpleSchemaChange struct {
	Action string `json:"action"`
	Name   string `json:"name"`
}

func simplifySchemaChanges(changes []*diff.SchemaDiff) []simpleSchemaChange {
	result := make([]simpleSchemaChange, len(changes))
	for i, c := range changes {
		result[i] = simpleSchemaChange{
			Action: string(c.Action),
			Name:   c.SchemaName,
		}
	}
	return result
}

type simpleTableChange struct {
	Action        string `json:"action"`
	Schema        string `json:"schema"`
	Name          string `json:"name"`
	ColumnChanges int    `json:"columnChanges"`
	IndexChanges  int    `json:"indexChanges"`
}

func simplifyTableChanges(changes []*diff.TableDiff) []simpleTableChange {
	result := make([]simpleTableChange, len(changes))
	for i, c := range changes {
		result[i] = simpleTableChange{
			Action:        string(c.Action),
			Schema:        c.SchemaName,
			Name:          c.TableName,
			ColumnChanges: len(c.ColumnChanges),
			IndexChanges:  len(c.IndexChanges),
		}
	}
	return result
}

type simpleViewChange struct {
	Action            string `json:"action"`
	Schema            string `json:"schema"`
	Name              string `json:"name"`
	DefinitionChanged bool   `json:"definitionChanged"`
}

func simplifyViewChanges(changes []*diff.ViewDiff) []simpleViewChange {
	result := make([]simpleViewChange, len(changes))
	for i, c := range changes {
		result[i] = simpleViewChange{
			Action:            string(c.Action),
			Schema:            c.SchemaName,
			Name:              c.ViewName,
			DefinitionChanged: c.DefinitionChanged,
		}
	}
	return result
}
