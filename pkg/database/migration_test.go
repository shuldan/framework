package database

import (
	"strings"
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
)

func TestBaseMigration(t *testing.T) {
	testNewMigration(t)
	testAddUpAndDown(t)
	testMultipleQueries(t)
}

func testNewMigration(t *testing.T) {
	t.Run("NewMigration", func(t *testing.T) {
		migration := NewMigration("001", "create users table")

		if migration.ID() != "001" {
			t.Errorf("expected ID '001', got '%s'", migration.ID())
		}
		if migration.Description() != "create users table" {
			t.Errorf("expected description 'create users table', got '%s'", migration.Description())
		}
		if len(migration.Up()) != 0 {
			t.Error("new migration should have empty up queries")
		}
		if len(migration.Down()) != 0 {
			t.Error("new migration should have empty down queries")
		}
	})
}

func testAddUpAndDown(t *testing.T) {
	t.Run("AddUp and AddDown", func(t *testing.T) {
		migration := NewMigration("001", "test")
		migration.AddUp("CREATE TABLE test (id INTEGER);")
		migration.AddDown("DROP TABLE test;")

		upQueries := migration.Up()
		downQueries := migration.Down()

		if len(upQueries) != 1 {
			t.Errorf("expected 1 up query, got %d", len(upQueries))
		}
		if upQueries[0] != "CREATE TABLE test (id INTEGER);" {
			t.Errorf("unexpected up query: %s", upQueries[0])
		}

		if len(downQueries) != 1 {
			t.Errorf("expected 1 down query, got %d", len(downQueries))
		}
		if downQueries[0] != "DROP TABLE test;" {
			t.Errorf("unexpected down query: %s", downQueries[0])
		}
	})
}

func testMultipleQueries(t *testing.T) {
	t.Run("Multiple queries", func(t *testing.T) {
		migration := NewMigration("002", "multiple operations")
		migration.AddUp("CREATE TABLE users (id INTEGER);")
		migration.AddUp("CREATE TABLE posts (id INTEGER);")
		migration.AddDown("DROP TABLE posts;")
		migration.AddDown("DROP TABLE users;")

		if len(migration.Up()) != 2 {
			t.Errorf("expected 2 up queries, got %d", len(migration.Up()))
		}
		if len(migration.Down()) != 2 {
			t.Errorf("expected 2 down queries, got %d", len(migration.Down()))
		}
	})
}

func TestMigrationBuilder(t *testing.T) {
	testCreateMigration(t)
	testCreateTable(t)
	testDropTable(t)
	testAddColumn(t)
	testCreateIndex(t)
	testCreateUniqueIndex(t)
	testAddForeignKey(t)
	testChainedOperations(t)
	testRawQueries(t)
}

func testCreateMigration(t *testing.T) {
	t.Run("CreateMigration", func(t *testing.T) {
		builder := CreateMigration("001", "test migration")
		if builder == nil {
			t.Fatal("CreateMigration returned nil")
		}
		if builder.migration == nil {
			t.Fatal("builder migration is nil")
		}
	})
}

func testCreateTable(t *testing.T) {
	t.Run("CreateTable", func(t *testing.T) {
		migration := CreateMigration("001", "create users table").
			CreateTable("users", "id INTEGER PRIMARY KEY", "name TEXT NOT NULL").
			Build()

		upQueries := migration.Up()
		downQueries := migration.Down()

		if len(upQueries) != 1 {
			t.Errorf("expected 1 up query, got %d", len(upQueries))
		}

		expectedUp := "CREATE TABLE users (\n    id INTEGER PRIMARY KEY,\n    name TEXT NOT NULL\n);"
		if upQueries[0] != expectedUp {
			t.Errorf("unexpected up query:\nexpected: %s\ngot: %s", expectedUp, upQueries[0])
		}

		if len(downQueries) != 1 {
			t.Errorf("expected 1 down query, got % d", len(downQueries))
		}

		expectedDown := "DROP TABLE IF EXISTS users;"
		if downQueries[0] != expectedDown {
			t.Errorf("unexpected down query:\nexpected: %s\ngot: %s", expectedDown, downQueries[0])
		}
	})
}

func testDropTable(t *testing.T) {
	t.Run("DropTable", func(t *testing.T) {
		migration := CreateMigration("002", "drop users table").
			DropTable("users").
			Build()

		upQueries := migration.Up()
		downQueries := migration.Down()

		expectedUp := "DROP TABLE IF EXISTS users;"
		if len(upQueries) != 1 || upQueries[0] != expectedUp {
			t.Errorf("unexpected up query: %v", upQueries)
		}

		if len(downQueries) != 1 || !strings.Contains(downQueries[0], "-- Cannot restore") {
			t.Errorf("unexpected down query: %v", downQueries)
		}
	})
}

func testAddColumn(t *testing.T) {
	t.Run("AddColumn", func(t *testing.T) {
		migration := CreateMigration("003", "add column").
			AddColumn("users", "email TEXT").
			Build()

		upQueries := migration.Up()
		downQueries := migration.Down()

		expectedUp := "ALTER TABLE users ADD COLUMN email TEXT;"
		if len(upQueries) != 1 || upQueries[0] != expectedUp {
			t.Errorf("unexpected up query: %v", upQueries)
		}

		expectedDown := "ALTER TABLE users DROP COLUMN email;"
		if len(downQueries) != 1 || downQueries[0] != expectedDown {
			t.Errorf("unexpected down query: %v", downQueries)
		}
	})
}

func testCreateIndex(t *testing.T) {
	t.Run("CreateIndex", func(t *testing.T) {
		migration := CreateMigration("004", "create index").
			CreateIndex("idx_users_email", "users", "email").
			Build()

		upQueries := migration.Up()
		downQueries := migration.Down()

		expectedUp := "CREATE INDEX idx_users_email ON users (email);"
		if len(upQueries) != 1 || upQueries[0] != expectedUp {
			t.Errorf("unexpected up query: %v", upQueries)
		}

		expectedDown := "DROP INDEX IF EXISTS idx_users_email;"
		if len(downQueries) != 1 || downQueries[0] != expectedDown {
			t.Errorf("unexpected down query: %v", downQueries)
		}
	})
}

func testCreateUniqueIndex(t *testing.T) {
	t.Run("CreateUniqueIndex", func(t *testing.T) {
		migration := CreateMigration("005", "create unique index").
			CreateUniqueIndex("idx_users_email_unique", "users", "email").
			Build()

		upQueries := migration.Up()
		expectedUp := "CREATE UNIQUE INDEX idx_users_email_unique ON users (email);"

		if len(upQueries) != 1 || upQueries[0] != expectedUp {
			t.Errorf("unexpected up query: %v", upQueries)
		}
	})
}

func testAddForeignKey(t *testing.T) {
	t.Run("AddForeignKey", func(t *testing.T) {
		migration := CreateMigration("006", "add foreign key").
			AddForeignKey("posts", "user_id", "users", "id").
			Build()

		upQueries := migration.Up()
		downQueries := migration.Down()

		expectedUp := "ALTER TABLE posts ADD CONSTRAINT fk_posts_user_id FOREIGN KEY (user_id) REFERENCES users(id);"
		if len(upQueries) != 1 || upQueries[0] != expectedUp {
			t.Errorf("unexpected up query: %v", upQueries)
		}

		expectedDown := "ALTER TABLE posts DROP CONSTRAINT IF EXISTS fk_posts_user_id;"
		if len(downQueries) != 1 || downQueries[0] != expectedDown {
			t.Errorf("unexpected down query: %v", downQueries)
		}
	})
}

func testChainedOperations(t *testing.T) {
	t.Run("ChainedOperations", func(t *testing.T) {
		migration := CreateMigration("007", "complex migration").
			CreateTable("categories", "id INTEGER PRIMARY KEY", "name TEXT NOT NULL").
			CreateIndex("idx_categories_name", "categories", "name").
			AddColumn("posts", "category_id INTEGER").
			AddForeignKey("posts", "category_id", "categories", "id").
			Build()

		upQueries := migration.Up()
		downQueries := migration.Down()

		if len(upQueries) != 4 {
			t.Errorf("expected 4 up queries, got %d", len(upQueries))
		}
		if len(downQueries) != 4 {
			t.Errorf("expected 4 down queries, got %d", len(downQueries))
		}

		if !strings.Contains(upQueries[0], "CREATE TABLE categories") {
			t.Errorf("first up query should create categories table: %s", upQueries[0])
		}

		if !strings.Contains(downQueries[len(downQueries)-1], "DROP TABLE IF EXISTS categories") {
			t.Errorf("last down query should drop categories table: %s", downQueries[len(downQueries)-1])
		}
	})
}

func testRawQueries(t *testing.T) {
	t.Run("RawQueries", func(t *testing.T) {
		migration := CreateMigration("008", "raw queries").
			RawUp("INSERT INTO settings (key, value) VALUES ('version', '1.0');").
			RawDown("DELETE FROM settings WHERE key = 'version';").
			Raw("CREATE TRIGGER test_trigger BEFORE INSERT ON users FOR EACH ROW BEGIN UPDATE counters SET value = value + 1; END;",
				"DROP TRIGGER IF EXISTS test_trigger;").
			Build()

		upQueries := migration.Up()
		downQueries := migration.Down()

		if len(upQueries) != 2 {
			t.Errorf("expected 2 up queries, got %d", len(upQueries))
		}
		if len(downQueries) != 2 {
			t.Errorf("expected 2 down queries, got %d", len(downQueries))
		}

		if !strings.Contains(upQueries[0], "INSERT INTO settings") {
			t.Errorf("first up query incorrect: %s", upQueries[0])
		}
		if !strings.Contains(upQueries[1], "CREATE TRIGGER") {
			t.Errorf("second up query incorrect: %s", upQueries[1])
		}
	})
}

func TestMigrationInterface(t *testing.T) {
	var _ contracts.Migration = (*BaseMigration)(nil)
	const id = "test"

	migration := NewMigration(id, "test migration")
	migration.AddUp("CREATE TABLE test (id INTEGER);")
	migration.AddDown("DROP TABLE test;")

	if migration.ID() != id {
		t.Error("ID method failed")
	}
	if migration.Description() != "test migration" {
		t.Error("Description method failed")
	}
	if len(migration.Up()) != 1 {
		t.Error("Up method failed")
	}
	if len(migration.Down()) != 1 {
		t.Error("Down method failed")
	}
}
