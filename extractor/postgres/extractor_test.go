package postgres

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestPostgreSQLExtractor(t *testing.T) {
	// Disable Ryuk container for environments with network restrictions
	t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	require.NoError(t, err)
	defer func() {
		if err := testcontainers.TerminateContainer(pgContainer); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}()

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	require.NoError(t, err)
	defer db.Close()

	// Wait for database to be ready
	err = db.Ping()
	require.NoError(t, err)

	// Create test schema
	setupTestSchema(t, db)

	// Create extractor
	extractor := NewExtractor(db, "testdb")

	// Test ListDatabases
	t.Run("ListDatabases", func(t *testing.T) {
		databases, err := extractor.ListDatabases(ctx)
		require.NoError(t, err)
		assert.Contains(t, databases, "testdb")
		// Should not include template databases
		assert.NotContains(t, databases, "template0")
		assert.NotContains(t, databases, "template1")
	})

	// Test ExtractSchema
	t.Run("ExtractSchema", func(t *testing.T) {
		schema, err := extractor.ExtractSchema(ctx)
		require.NoError(t, err)
		require.NotNil(t, schema)

		// Verify database metadata
		assert.Equal(t, "testdb", schema.Name)
		assert.Contains(t, schema.SearchPath, "public")
		assert.GreaterOrEqual(t, len(schema.Schemas), 2, "should have at least public and app schemas")

		// Find public schema
		var publicSchema *struct {
			Name              string
			Tables            int
			Views             int
			MaterializedViews int
			Functions         int
			Procedures        int
			Sequences         int
			Extensions        int
			EnumTypes         int
		}

		for _, s := range schema.Schemas {
			if s.Name == "public" {
				publicSchema = &struct {
					Name              string
					Tables            int
					Views             int
					MaterializedViews int
					Functions         int
					Procedures        int
					Sequences         int
					Extensions        int
					EnumTypes         int
				}{
					Name:              s.Name,
					Tables:            len(s.Tables),
					Views:             len(s.Views),
					MaterializedViews: len(s.MaterializedViews),
					Functions:         len(s.Functions),
					Procedures:        len(s.Procedures),
					Sequences:         len(s.Sequences),
					Extensions:        len(s.Extensions),
					EnumTypes:         len(s.EnumTypes),
				}
				break
			}
		}
		require.NotNil(t, publicSchema, "public schema should exist")

		// Verify public schema contents
		assert.GreaterOrEqual(t, publicSchema.Tables, 3, "should have at least 3 tables")
		assert.GreaterOrEqual(t, publicSchema.Views, 1, "should have at least 1 view")
		assert.GreaterOrEqual(t, publicSchema.MaterializedViews, 1, "should have at least 1 materialized view")
		assert.GreaterOrEqual(t, publicSchema.Functions, 1, "should have at least 1 function")
		assert.GreaterOrEqual(t, publicSchema.Procedures, 1, "should have at least 1 procedure")
		assert.GreaterOrEqual(t, publicSchema.Sequences, 1, "should have at least 1 sequence")
		assert.GreaterOrEqual(t, publicSchema.EnumTypes, 1, "should have at least 1 enum type")

		// Find app schema
		var appSchema *struct {
			Name   string
			Tables int
		}

		for _, s := range schema.Schemas {
			if s.Name == "app" {
				appSchema = &struct {
					Name   string
					Tables int
				}{
					Name:   s.Name,
					Tables: len(s.Tables),
				}
				break
			}
		}
		require.NotNil(t, appSchema, "app schema should exist")
		assert.GreaterOrEqual(t, appSchema.Tables, 1, "app schema should have at least 1 table")
	})

	// Test users table details
	t.Run("UsersTable", func(t *testing.T) {
		schema, err := extractor.ExtractSchema(ctx)
		require.NoError(t, err)

		var usersTable *struct {
			Name             string
			Columns          int
			Indexes          int
			ForeignKeys      int
			CheckConstraints int
			Comment          string
		}

		for _, s := range schema.Schemas {
			if s.Name == "public" {
				for _, table := range s.Tables {
					if table.Name == "users" {
						usersTable = &struct {
							Name             string
							Columns          int
							Indexes          int
							ForeignKeys      int
							CheckConstraints int
							Comment          string
						}{
							Name:             table.Name,
							Columns:          len(table.Columns),
							Indexes:          len(table.Indexes),
							ForeignKeys:      len(table.ForeignKeys),
							CheckConstraints: len(table.CheckConstraints),
							Comment:          table.Comment,
						}
						break
					}
				}
				break
			}
		}

		require.NotNil(t, usersTable, "users table should exist")
		assert.Equal(t, 6, usersTable.Columns, "users should have 6 columns")
		assert.GreaterOrEqual(t, usersTable.Indexes, 2, "users should have at least 2 indexes")
		assert.Equal(t, "User accounts", usersTable.Comment)
	})

	// Test foreign keys
	t.Run("ForeignKeys", func(t *testing.T) {
		schema, err := extractor.ExtractSchema(ctx)
		require.NoError(t, err)

		var postsTable *struct {
			ForeignKeys int
			FKName      string
		}

		for _, s := range schema.Schemas {
			if s.Name == "public" {
				for _, table := range s.Tables {
					if table.Name == "posts" {
						postsTable = &struct {
							ForeignKeys int
							FKName      string
						}{
							ForeignKeys: len(table.ForeignKeys),
						}
						if len(table.ForeignKeys) > 0 {
							postsTable.FKName = table.ForeignKeys[0].Name
						}
						break
					}
				}
				break
			}
		}

		require.NotNil(t, postsTable, "posts table should exist")
		assert.GreaterOrEqual(t, postsTable.ForeignKeys, 1, "posts should have at least 1 foreign key")
		assert.NotEmpty(t, postsTable.FKName, "foreign key should have a name")
	})

	// Test indexes
	t.Run("Indexes", func(t *testing.T) {
		schema, err := extractor.ExtractSchema(ctx)
		require.NoError(t, err)

		var usersTable *struct {
			Indexes       int
			HasPrimaryKey bool
			HasEmailIndex bool
		}

		for _, s := range schema.Schemas {
			if s.Name == "public" {
				for _, table := range s.Tables {
					if table.Name == "users" {
						usersTable = &struct {
							Indexes       int
							HasPrimaryKey bool
							HasEmailIndex bool
						}{
							Indexes: len(table.Indexes),
						}
						for _, idx := range table.Indexes {
							if idx.Primary {
								usersTable.HasPrimaryKey = true
							}
							if idx.Name == "idx_users_email" {
								usersTable.HasEmailIndex = true
							}
						}
						break
					}
				}
				break
			}
		}

		require.NotNil(t, usersTable, "users table should exist")
		assert.True(t, usersTable.HasPrimaryKey, "users should have PRIMARY key")
		assert.True(t, usersTable.HasEmailIndex, "users should have idx_users_email index")
	})

	// Test views
	t.Run("Views", func(t *testing.T) {
		schema, err := extractor.ExtractSchema(ctx)
		require.NoError(t, err)

		var activeUsersView *struct {
			Name       string
			Definition string
		}

		for _, s := range schema.Schemas {
			if s.Name == "public" {
				for _, view := range s.Views {
					if view.Name == "active_users" {
						activeUsersView = &struct {
							Name       string
							Definition string
						}{
							Name:       view.Name,
							Definition: view.Definition,
						}
						break
					}
				}
				break
			}
		}

		require.NotNil(t, activeUsersView, "active_users view should exist")
		assert.NotEmpty(t, activeUsersView.Definition, "view should have definition")
	})

	// Test materialized views
	t.Run("MaterializedViews", func(t *testing.T) {
		schema, err := extractor.ExtractSchema(ctx)
		require.NoError(t, err)

		var userStatsMV *struct {
			Name       string
			Definition string
		}

		for _, s := range schema.Schemas {
			if s.Name == "public" {
				for _, mv := range s.MaterializedViews {
					if mv.Name == "user_stats" {
						userStatsMV = &struct {
							Name       string
							Definition string
						}{
							Name:       mv.Name,
							Definition: mv.Definition,
						}
						break
					}
				}
				break
			}
		}

		require.NotNil(t, userStatsMV, "user_stats materialized view should exist")
		assert.NotEmpty(t, userStatsMV.Definition, "materialized view should have definition")
	})

	// Test sequences
	t.Run("Sequences", func(t *testing.T) {
		schema, err := extractor.ExtractSchema(ctx)
		require.NoError(t, err)

		var usersSeq *struct {
			Name      string
			DataType  string
			Start     int64
			Increment int64
		}

		for _, s := range schema.Schemas {
			if s.Name == "public" {
				for _, seq := range s.Sequences {
					if seq.Name == "users_id_seq" {
						usersSeq = &struct {
							Name      string
							DataType  string
							Start     int64
							Increment int64
						}{
							Name:      seq.Name,
							DataType:  seq.DataType,
							Start:     seq.Start,
							Increment: seq.Increment,
						}
						break
					}
				}
				break
			}
		}

		require.NotNil(t, usersSeq, "users_id_seq sequence should exist")
		assert.Equal(t, "bigint", usersSeq.DataType)
		assert.Equal(t, int64(1), usersSeq.Increment)
	})

	// Test enum types
	t.Run("EnumTypes", func(t *testing.T) {
		schema, err := extractor.ExtractSchema(ctx)
		require.NoError(t, err)

		var statusEnum *struct {
			Name   string
			Values []string
		}

		for _, s := range schema.Schemas {
			if s.Name == "public" {
				for _, enum := range s.EnumTypes {
					if enum.Name == "user_status" {
						statusEnum = &struct {
							Name   string
							Values []string
						}{
							Name:   enum.Name,
							Values: enum.Values,
						}
						break
					}
				}
				break
			}
		}

		require.NotNil(t, statusEnum, "user_status enum should exist")
		assert.Contains(t, statusEnum.Values, "active")
		assert.Contains(t, statusEnum.Values, "inactive")
		assert.Contains(t, statusEnum.Values, "suspended")
	})

	// Test functions
	t.Run("Functions", func(t *testing.T) {
		schema, err := extractor.ExtractSchema(ctx)
		require.NoError(t, err)

		var getUserNameFunc *struct {
			Name       string
			Definition string
		}

		for _, s := range schema.Schemas {
			if s.Name == "public" {
				for _, fn := range s.Functions {
					if fn.Name == "get_user_name" {
						getUserNameFunc = &struct {
							Name       string
							Definition string
						}{
							Name:       fn.Name,
							Definition: fn.Definition,
						}
						break
					}
				}
				break
			}
		}

		require.NotNil(t, getUserNameFunc, "get_user_name function should exist")
		assert.NotEmpty(t, getUserNameFunc.Definition, "function should have definition")
	})

	// Test procedures
	t.Run("Procedures", func(t *testing.T) {
		schema, err := extractor.ExtractSchema(ctx)
		require.NoError(t, err)

		var updateUserProc *struct {
			Name       string
			Definition string
		}

		for _, s := range schema.Schemas {
			if s.Name == "public" {
				for _, proc := range s.Procedures {
					if proc.Name == "update_user_email" {
						updateUserProc = &struct {
							Name       string
							Definition string
						}{
							Name:       proc.Name,
							Definition: proc.Definition,
						}
						break
					}
				}
				break
			}
		}

		require.NotNil(t, updateUserProc, "update_user_email procedure should exist")
		assert.NotEmpty(t, updateUserProc.Definition, "procedure should have definition")
	})

	// Test multi-schema support
	t.Run("MultiSchema", func(t *testing.T) {
		schema, err := extractor.ExtractSchema(ctx)
		require.NoError(t, err)

		schemaNames := make([]string, len(schema.Schemas))
		for i, s := range schema.Schemas {
			schemaNames[i] = s.Name
		}

		assert.Contains(t, schemaNames, "public")
		assert.Contains(t, schemaNames, "app")
	})
}

func setupTestSchema(t *testing.T, db *sql.DB) {
	queries := []string{
		// Create enum type
		`CREATE TYPE user_status AS ENUM ('active', 'inactive', 'suspended')`,

		// Create users table with sequence
		`CREATE TABLE users (
			id BIGSERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(100) NOT NULL,
			status user_status DEFAULT 'active',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,

		`COMMENT ON TABLE users IS 'User accounts'`,
		`CREATE INDEX idx_users_email ON users(email)`,

		// Create posts table with foreign key
		`CREATE TABLE posts (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title VARCHAR(255) NOT NULL,
			content TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE INDEX idx_posts_user_id ON posts(user_id)`,

		// Create comments table
		`CREATE TABLE comments (
			id BIGSERIAL PRIMARY KEY,
			post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
			author VARCHAR(100) NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create view
		`CREATE VIEW active_users AS
		SELECT id, email, name, created_at
		FROM users
		WHERE status = 'active' AND created_at >= NOW() - INTERVAL '30 days'`,

		// Create materialized view
		`CREATE MATERIALIZED VIEW user_stats AS
		SELECT
			u.id,
			u.name,
			COUNT(p.id) AS post_count,
			MAX(p.created_at) AS last_post_at
		FROM users u
		LEFT JOIN posts p ON u.id = p.user_id
		GROUP BY u.id, u.name`,

		// Create function
		`CREATE FUNCTION get_user_name(user_id BIGINT)
		RETURNS VARCHAR(100) AS $$
		DECLARE
			user_name VARCHAR(100);
		BEGIN
			SELECT name INTO user_name FROM users WHERE id = user_id;
			RETURN user_name;
		END;
		$$ LANGUAGE plpgsql`,

		// Create procedure (PostgreSQL 11+)
		`CREATE PROCEDURE update_user_email(p_user_id BIGINT, p_email VARCHAR(255))
		LANGUAGE plpgsql AS $$
		BEGIN
			UPDATE users SET email = p_email, updated_at = NOW() WHERE id = p_user_id;
		END;
		$$`,

		// Create trigger function
		`CREATE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = NOW();
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,

		// Create trigger
		`CREATE TRIGGER update_users_updated_at
		BEFORE UPDATE ON users
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column()`,

		// Create app schema
		`CREATE SCHEMA app`,

		// Create table in app schema
		`CREATE TABLE app.settings (
			key VARCHAR(100) PRIMARY KEY,
			value TEXT,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,

		// Insert test data
		`INSERT INTO users (email, name, status) VALUES
			('alice@example.com', 'Alice', 'active'),
			('bob@example.com', 'Bob', 'active'),
			('charlie@example.com', 'Charlie', 'inactive')`,

		`INSERT INTO posts (user_id, title, content) VALUES
			(1, 'First Post', 'This is my first post'),
			(1, 'Second Post', 'Another great post'),
			(2, 'Bob Post', 'Hello from Bob')`,

		`INSERT INTO comments (post_id, author, content) VALUES
			(1, 'Bob', 'Great post!'),
			(1, 'Charlie', 'Thanks for sharing')`,

		`INSERT INTO app.settings (key, value) VALUES
			('theme', 'dark'),
			('language', 'en')`,
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		require.NoError(t, err, "failed to execute query: %s", query)
	}
}
