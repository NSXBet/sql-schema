package mysql

import (
	"github.com/nsxbet/sql-schema/comparer/engine"
)

func init() {
	// Register MySQL comparer on package initialization
	engine.Register(engine.MySQL, NewComparer())
	// MariaDB and TiDB use the same comparer
	engine.Register(engine.MariaDB, NewComparer())
	engine.Register(engine.TiDB, NewComparer())
	engine.Register(engine.OceanBase, NewComparer())
}
