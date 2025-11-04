# Schema Extraction Examples

This directory contains example programs demonstrating how to use the `schemaextract` package.

## Prerequisites

### MySQL Testing
- MySQL 5.7+ or MySQL 8.0+ server running
- A test database with some tables, views, functions, etc.

### PostgreSQL Testing
- PostgreSQL 10+ server running
- A test database with some tables, views, functions, etc.

## Running the Examples

### MySQL Example

1. Set the connection string:
```bash
export MYSQL_DSN="user:password@tcp(localhost:3306)/your_database"
```

2. Run the example:
```bash
cd examples/mysql
go run main.go
```

3. Check the output:
- Console output shows schema summary
- `mysql_schema.json` contains the full schema in JSON format

### PostgreSQL Example

1. Set the connection string:
```bash
export PG_DSN="postgres://user:password@localhost:5432/your_database?sslmode=disable"
```

2. Run the example:
```bash
cd examples/postgres
go run main.go
```

3. Check the output:
- Console output shows schema summary
- `postgres_schema.json` contains the full schema in JSON format

## Example Output

### MySQL
```
Connecting to MySQL database: test_db
✓ Connected to MySQL database

--- Listing Databases ---
Found 3 databases:
  - test_db
  - app_db
  - analytics_db

--- Extracting Schema for 'test_db' ---

=== Schema Summary ===
Database: test_db
Character Set: utf8mb4
Collation: utf8mb4_unicode_ci
Schemas: 1

Schema:  (unnamed schema in MySQL)
  Tables: 5
  Views: 2
  Functions: 3
  Procedures: 1
  Events: 0

  Tables:
    - users (8 columns, 3 indexes, 0 foreign keys)
      Engine: InnoDB, Rows: 1543, Size: 163840 bytes
      Columns:
        - id bigint NOT NULL DEFAULT AUTO_INCREMENT
        - email varchar(255) NOT NULL
        - created_at timestamp NULL DEFAULT CURRENT_TIMESTAMP
        ...
```

### PostgreSQL
```
Connecting to PostgreSQL database: test_db
✓ Connected to PostgreSQL database

--- Listing Databases ---
Found 2 databases:
  - test_db
  - app_db

--- Extracting Schema for 'test_db' ---

=== Schema Summary ===
Database: test_db
Search Path: [public]
Schemas: 1

Schema: public
  Tables: 8
  Views: 3
  Materialized Views: 1
  Functions: 5
  Procedures: 2
  Sequences: 4
  Extensions: 2
  Enum Types: 1

  Tables:
    - users (10 columns, 4 indexes, 1 foreign keys)
      Data Size: 245760 bytes, Index Size: 98304 bytes
      Columns:
        - id bigint NOT NULL DEFAULT nextval('users_id_seq'::regclass)
        - email character varying(255) NOT NULL
        - created_at timestamp with time zone NULL DEFAULT now()
        ...
```

## Testing with the Bytebase Database

If you want to test with an actual Bytebase database:

### MySQL (Bytebase uses PostgreSQL by default, but if you have MySQL)
```bash
export MYSQL_DSN="root@tcp(localhost:3306)/bytebase"
cd examples/mysql
go run main.go
```

### PostgreSQL (Default for Bytebase)
```bash
export PG_DSN="postgres://bbdev@localhost/bbdev?sslmode=disable"
cd examples/postgres
go run main.go
```

## What Gets Extracted

### MySQL
- ✅ Tables with columns, indexes, foreign keys, check constraints
- ✅ Views with definitions
- ✅ Functions and stored procedures
- ✅ Triggers
- ✅ Events
- ✅ Partitions (all types: RANGE, LIST, HASH, KEY)
- ✅ Generated columns (VIRTUAL/STORED)
- ✅ Character sets and collations
- ✅ Table statistics (row count, data size, index size)

### PostgreSQL
- ✅ Schemas (multiple schemas per database)
- ✅ Tables with columns, indexes, foreign keys, check constraints
- ✅ Views and materialized views
- ✅ Functions and procedures
- ✅ Triggers
- ✅ Sequences with all properties
- ✅ Extensions
- ✅ Enum types
- ✅ Partitions (PostgreSQL 10+)
- ✅ Generated columns
- ✅ Search path
- ✅ Table and index sizes

## Troubleshooting

### MySQL Connection Issues
- Make sure MySQL server is running
- Check that the user has appropriate permissions (at least SELECT on all tables in information_schema)
- Try testing the connection: `mysql -u user -p -h localhost`

### PostgreSQL Connection Issues
- Make sure PostgreSQL server is running
- Check that the user has appropriate permissions
- Try testing the connection: `psql -U user -h localhost -d dbname`
- If using SSL, adjust the `sslmode` parameter in the DSN

### No Tables Found
- Make sure you're connecting to the correct database
- Check that the database has tables
- Verify permissions: the user needs SELECT privilege on system tables

### JSON Export Errors
- Check that you have write permissions in the current directory
- Make sure there's enough disk space
