package engine

// Engine represents a database engine type.
type Engine string

const (
	// MySQL represents MySQL database engine.
	MySQL Engine = "MYSQL"
	// PostgreSQL represents PostgreSQL database engine.
	PostgreSQL Engine = "POSTGRES"
	// MariaDB represents MariaDB database engine.
	MariaDB Engine = "MARIADB"
	// TiDB represents TiDB database engine.
	TiDB Engine = "TIDB"
	// OceanBase represents OceanBase database engine.
	OceanBase Engine = "OCEANBASE"
	// Unknown represents an unknown database engine.
	Unknown Engine = "UNKNOWN"
)

// String returns the string representation of the engine.
func (e Engine) String() string {
	return string(e)
}

// IsMySQL returns true if the engine is MySQL-like (MySQL, MariaDB, TiDB, OceanBase).
func (e Engine) IsMySQL() bool {
	switch e {
	case MySQL, MariaDB, TiDB, OceanBase:
		return true
	default:
		return false
	}
}

// IsPostgreSQL returns true if the engine is PostgreSQL.
func (e Engine) IsPostgreSQL() bool {
	return e == PostgreSQL
}

// IsCaseSensitive returns true if the engine is case-sensitive for object names.
// MySQL, MariaDB, TiDB, and OceanBase are case-insensitive for column names and other details.
func (e Engine) IsCaseSensitive() bool {
	switch e {
	case MySQL, MariaDB, TiDB, OceanBase:
		return false
	default:
		return true
	}
}

// ParseEngine parses a string into an Engine type.
func ParseEngine(s string) Engine {
	switch s {
	case "MYSQL", "mysql", "MySQL":
		return MySQL
	case "POSTGRES", "postgres", "PostgreSQL", "POSTGRESQL", "postgresql":
		return PostgreSQL
	case "MARIADB", "mariadb", "MariaDB":
		return MariaDB
	case "TIDB", "tidb", "TiDB":
		return TiDB
	case "OCEANBASE", "oceanbase", "OceanBase":
		return OceanBase
	default:
		return Unknown
	}
}
