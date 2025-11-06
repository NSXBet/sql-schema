package mysql

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
)

func TestMySQLExtractor(t *testing.T) {
	// Disable Ryuk container for environments with network restrictions
	t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	ctx := context.Background()

	// Start MySQL container with log_bin_trust_function_creators enabled
	mysqlContainer, err := mysql.Run(ctx,
		"mysql:8.0",
		mysql.WithDatabase("testdb"),
		mysql.WithUsername("testuser"),
		mysql.WithPassword("testpass"),
		testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Cmd: []string{"--log_bin_trust_function_creators=1"},
			},
		}),
	)
	require.NoError(t, err)
	defer func() {
		if err := testcontainers.TerminateContainer(mysqlContainer); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}()

	// Get connection string
	connStr, err := mysqlContainer.ConnectionString(ctx)
	require.NoError(t, err)

	// Connect to database
	db, err := sql.Open("mysql", connStr)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

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
	})

	// Test ExtractSchema
	t.Run("ExtractSchema", func(t *testing.T) {
		schema, err := extractor.ExtractSchema(ctx)
		require.NoError(t, err)
		require.NotNil(t, schema)

		// Verify database metadata
		assert.Equal(t, "testdb", schema.Name)
		assert.NotEmpty(t, schema.CharacterSet)
		assert.NotEmpty(t, schema.Collation)
		assert.Len(t, schema.Schemas, 1)

		s := schema.Schemas[0]
		assert.Equal(t, "", s.Name) // MySQL has unnamed schema

		// Verify tables
		assert.GreaterOrEqual(t, len(s.Tables), 3, "should have at least 3 tables")

		// Find and verify users table
		var usersTable *struct {
			Name             string
			Columns          int
			Indexes          int
			ForeignKeys      int
			CheckConstraints int
		}
		for _, table := range s.Tables {
			if table.Name == "users" {
				usersTable = &struct {
					Name             string
					Columns          int
					Indexes          int
					ForeignKeys      int
					CheckConstraints int
				}{
					Name:             table.Name,
					Columns:          len(table.Columns),
					Indexes:          len(table.Indexes),
					ForeignKeys:      len(table.ForeignKeys),
					CheckConstraints: len(table.CheckConstraints),
				}
				break
			}
		}
		require.NotNil(t, usersTable, "users table should exist")
		assert.Equal(t, 5, usersTable.Columns, "users should have 5 columns")
		assert.GreaterOrEqual(t, usersTable.Indexes, 1, "users should have at least 1 index (PRIMARY)")

		// Verify views
		assert.GreaterOrEqual(t, len(s.Views), 1, "should have at least 1 view")
		var activeUsersView bool
		for _, view := range s.Views {
			if view.Name == "active_users" {
				activeUsersView = true
				assert.NotEmpty(t, view.Definition)
				break
			}
		}
		assert.True(t, activeUsersView, "active_users view should exist")

		// Verify functions
		assert.GreaterOrEqual(t, len(s.Functions), 1, "should have at least 1 function")
		var getUserNameFunc bool
		for _, fn := range s.Functions {
			if fn.Name == "get_user_name" {
				getUserNameFunc = true
				assert.NotEmpty(t, fn.Definition)
				break
			}
		}
		assert.True(t, getUserNameFunc, "get_user_name function should exist")

		// Verify procedures
		assert.GreaterOrEqual(t, len(s.Procedures), 1, "should have at least 1 procedure")
		var updateUserProc bool
		for _, proc := range s.Procedures {
			if proc.Name == "update_user_email" {
				updateUserProc = true
				assert.NotEmpty(t, proc.Definition)
				break
			}
		}
		assert.True(t, updateUserProc, "update_user_email procedure should exist")
	})

	// Test foreign keys
	t.Run("ForeignKeys", func(t *testing.T) {
		schema, err := extractor.ExtractSchema(ctx)
		require.NoError(t, err)

		var postsTable *struct {
			ForeignKeys int
			FKName      string
		}
		for _, table := range schema.Schemas[0].Tables {
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
		require.NotNil(t, postsTable, "posts table should exist")
		assert.GreaterOrEqual(t, postsTable.ForeignKeys, 1, "posts should have at least 1 foreign key")
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
		for _, table := range schema.Schemas[0].Tables {
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
					if idx.Name == "idx_email" {
						usersTable.HasEmailIndex = true
					}
				}
				break
			}
		}
		require.NotNil(t, usersTable, "users table should exist")
		assert.True(t, usersTable.HasPrimaryKey, "users should have PRIMARY key")
		assert.True(t, usersTable.HasEmailIndex, "users should have idx_email index")
	})
}

func setupTestSchema(t *testing.T, db *sql.DB) {
	queries := []string{
		// Create users table
		`CREATE TABLE users (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			name VARCHAR(100) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_email (email)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='User accounts'`,

		// Create posts table with foreign key
		`CREATE TABLE posts (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_id BIGINT NOT NULL,
			title VARCHAR(255) NOT NULL,
			content TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			INDEX idx_user_id (user_id)
		) ENGINE=InnoDB`,

		// Create comments table
		`CREATE TABLE comments (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			post_id BIGINT NOT NULL,
			author VARCHAR(100) NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
		) ENGINE=InnoDB`,

		// Create view
		`CREATE VIEW active_users AS
		SELECT id, email, name, created_at
		FROM users
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)`,

		// Create function
		`CREATE FUNCTION get_user_name(user_id BIGINT)
		RETURNS VARCHAR(100)
		DETERMINISTIC
		READS SQL DATA
		BEGIN
			DECLARE user_name VARCHAR(100);
			SELECT name INTO user_name FROM users WHERE id = user_id;
			RETURN user_name;
		END`,

		// Create procedure
		`CREATE PROCEDURE update_user_email(IN p_user_id BIGINT, IN p_email VARCHAR(255))
		BEGIN
			UPDATE users SET email = p_email, updated_at = NOW() WHERE id = p_user_id;
		END`,

		// Insert test data
		`INSERT INTO users (email, name) VALUES
			('alice@example.com', 'Alice'),
			('bob@example.com', 'Bob'),
			('charlie@example.com', 'Charlie')`,

		`INSERT INTO posts (user_id, title, content) VALUES
			(1, 'First Post', 'This is my first post'),
			(1, 'Second Post', 'Another great post'),
			(2, 'Bob Post', 'Hello from Bob')`,

		`INSERT INTO comments (post_id, author, content) VALUES
			(1, 'Bob', 'Great post!'),
			(1, 'Charlie', 'Thanks for sharing')`,
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		require.NoError(t, err, "failed to execute query: %s", query)
	}
}
