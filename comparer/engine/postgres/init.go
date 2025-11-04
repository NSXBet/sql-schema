package postgres

import (
	"github.com/nsxbet/sql-schema/comparer/engine"
)

func init() {
	// Register PostgreSQL comparer on package initialization
	engine.Register(engine.PostgreSQL, NewComparer())
}
