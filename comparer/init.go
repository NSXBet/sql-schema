package comparer

import (
	// Auto-import engine implementations to register comparers.
	// This allows users to use the comparison APIs without manually
	// importing engine packages.
	_ "github.com/nsxbet/sql-schema/comparer/engine/mysql"
	_ "github.com/nsxbet/sql-schema/comparer/engine/postgres"
)

// This file ensures that all database engine comparers are automatically
// registered when the util package is imported. Users no longer need to
// manually import engine packages with blank imports.
//
// Example usage:
//
//	import (
//	    "github.com/nsxbet/sql-schema/comparer"
//	    "github.com/nsxbet/sql-schema/comparer/engine"
//	)
//
//	// Engines are already registered, no need for:
//	// _ "github.com/nsxbet/sql-schema/comparer/engine/postgres"
//	// _ "github.com/nsxbet/sql-schema/comparer/engine/mysql"
//
//	opts := &util.CompareOptions{Engine: engine.PostgreSQL}
//	diff, err := util.CompareSchemasDetailed(oldSchema, newSchema, opts)
