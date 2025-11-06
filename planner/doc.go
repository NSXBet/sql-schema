// Package planner provides migration planning and strategy generation
// for schema changes.
//
// # Overview
//
// The planner package analyzes schema differences and generates ordered
// migration plans that respect database constraints and dependencies.
//
// # Features
//
//   - Dependency resolution (e.g., drop foreign keys before dropping tables)
//   - Safe operation ordering (e.g., add column before adding constraint)
//   - Destructive operation detection (data loss warnings)
//   - Rollback plan generation
//   - Multi-step migration strategies
//
// # Usage
//
// Generate a migration plan:
//
//	diffs, _ := comparer.CompareSchemas(oldSchema, newSchema, nil)
//	plan, err := planner.GeneratePlan(diffs, &planner.Options{
//	    AllowDestructive: false,
//	    GenerateRollback: true,
//	})
//
//	for _, step := range plan.Steps {
//	    fmt.Printf("Step %d: %s\n", step.Order, step.Description)
//	    if step.Destructive {
//	        fmt.Println("  WARNING: This step may cause data loss")
//	    }
//	}
//
// # Migration Strategies
//
// The planner supports different migration strategies:
//   - Safe: Only non-destructive changes
//   - Standard: Includes destructive changes with warnings
//   - Aggressive: Optimizes for speed, may cause downtime
//
// # Operation Ordering
//
// The planner ensures operations are executed in the correct order:
//
//  1. Drop foreign keys
//  2. Drop indexes
//  3. Modify/drop columns
//  4. Add/modify tables
//  5. Add columns
//  6. Add indexes
//  7. Add foreign keys
//
// # Rollback Planning
//
// When enabled, the planner generates rollback steps that can undo
// non-destructive changes. Destructive operations (DROP TABLE, DROP COLUMN)
// cannot be automatically rolled back.
//
// # Limitations
//
// Current limitations:
//   - Does not handle data migrations (only DDL)
//   - Cannot generate rollback for destructive operations
//   - Does not optimize for minimal downtime
package planner
