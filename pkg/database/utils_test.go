package database

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestQueryBuilder(t *testing.T) {
	t.Run("Simple SELECT", func(t *testing.T) {
		qb := NewQueryBuilder()
		query, args := qb.Select("id", "name").From("users").Build()

		expected := "SELECT id, name FROM users"
		if query != expected {
			t.Errorf("expected query '%s', got '%s'", expected, query)
		}
		if len(args) != 0 {
			t.Errorf("expected 0 args, got %d", len(args))
		}
	})

	t.Run("SELECT with WHERE", func(t *testing.T) {
		qb := NewQueryBuilder()
		query, args := qb.Select("*").From("users").Where("id = ?", 1).Build()

		expected := "SELECT * FROM users WHERE id = ?"
		if query != expected {
			t.Errorf("expected query '%s', got '%s'", expected, query)
		}
		if len(args) != 1 || args[0] != 1 {
			t.Errorf("expected args [1], got %v", args)
		}
	})

	t.Run("Complex query with JOIN and conditions", func(t *testing.T) {
		qb := NewQueryBuilder()
		query, args := qb.
			Select("u.name", "p.title").
			From("users u").
			Join("posts p", "p.user_id = u.id").
			Where("u.active = ?", true).
			And("p.published = ?", true).
			OrderBy("u.name", "ASC").
			Limit(10).
			Offset(0).
			Build()

		expected := "SELECT u.name, p.title FROM users u JOIN posts p ON p.user_id = u.id WHERE u.active = ? AND p.published = ? ORDER BY u.name ASC LIMIT ? OFFSET ?"
		if query != expected {
			t.Errorf("expected query '%s', got '%s'", expected, query)
		}

		expectedArgs := []interface{}{true, true, 10, 0}
		if len(args) != len(expectedArgs) {
			t.Errorf("expected %d args, got %d", len(expectedArgs), len(args))
		}
		for i, arg := range args {
			if arg != expectedArgs[i] {
				t.Errorf("expected arg %d to be %v, got %v", i, expectedArgs[i], arg)
			}
		}
	})

	t.Run("LEFT JOIN with GROUP BY and HAVING", func(t *testing.T) {
		qb := NewQueryBuilder()
		query, args := qb.
			Select("u.id", "COUNT(p.id) as post_count").
			From("users u").
			LeftJoin("posts p", "p.user_id = u.id").
			GroupBy("u.id").
			Having("COUNT(p.id) > ?", 5).
			Build()

		expected := "SELECT u.id, COUNT(p.id) as post_count FROM users u LEFT JOIN posts p ON p.user_id = u.id GROUP BY u.id HAVING COUNT(p.id) > ?"
		if query != expected {
			t.Errorf("expected query '%s', got '%s'", expected, query)
		}
		if len(args) != 1 || args[0] != 5 {
			t.Errorf("expected args [5], got %v", args)
		}
	})

	t.Run("OR condition", func(t *testing.T) {
		qb := NewQueryBuilder()
		query, args := qb.
			Select("*").
			From("users").
			Where("name = ?", "John").
			Or("email = ?", "john@example.com").
			Build()

		expected := "SELECT * FROM users WHERE name = ? OR email = ?"
		if query != expected {
			t.Errorf("expected query '%s', got '%s'", expected, query)
		}

		expectedArgs := []interface{}{"John", "john@example.com"}
		if len(args) != len(expectedArgs) {
			t.Errorf("expected %d args, got %d", len(expectedArgs), len(args))
		}
	})

	t.Run("Reset query builder", func(t *testing.T) {
		qb := NewQueryBuilder()
		qb.Select("*").From("users").Where("id = ?", 1)

		qb.Reset()
		query, args := qb.Select("name").From("posts").Build()

		expected := "SELECT name FROM posts"
		if query != expected {
			t.Errorf("expected query '%s' after reset, got '%s'", expected, query)
		}
		if len(args) != 0 {
			t.Errorf("expected 0 args after reset, got %d", len(args))
		}
	})
}

func TestTransactionManager(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tm := NewTransactionManager(db)

	t.Run("Successful transaction", func(t *testing.T) {

		_, err := db.Exec("CREATE TABLE test_tx (id INTEGER PRIMARY KEY, value TEXT)")
		if err != nil {
			t.Fatalf("failed to create test table: %v", err)
		}

		err = tm.Execute(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, "INSERT INTO test_tx (value) VALUES (?)", "test1")
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, "INSERT INTO test_tx (value) VALUES (?)", "test2")
			return err
		})

		if err != nil {
			t.Errorf("transaction execution failed: %v", err)
		}

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test_tx").Scan(&count)
		if err != nil {
			t.Errorf("failed to count records: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 records, got %d", count)
		}
	})

	t.Run("Failed transaction with rollback", func(t *testing.T) {
		initialCount := 0
		db.QueryRow("SELECT COUNT(*) FROM test_tx").Scan(&initialCount)

		err := tm.Execute(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, "INSERT INTO test_tx (value) VALUES (?)", "test3")
			if err != nil {
				return err
			}

			_, err = tx.ExecContext(ctx, "INSERT INTO invalid_table (value) VALUES (?)", "test4")
			return err
		})

		if err == nil {
			t.Error("expected transaction to fail")
		}

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test_tx").Scan(&count)
		if err != nil {
			t.Errorf("failed to count records: %v", err)
		}
		if count != initialCount {
			t.Errorf("expected %d records (no change), got %d", initialCount, count)
		}
	})

	t.Run("Panic recovery", func(t *testing.T) {
		initialCount := 0
		db.QueryRow("SELECT COUNT(*) FROM test_tx").Scan(&initialCount)

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic to be re-raised")
			}
		}()

		tm.Execute(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, "INSERT INTO test_tx (value) VALUES (?)", "panic_test")
			if err != nil {
				return err
			}
			panic("test panic")
		})
	})
}

func TestBatchProcessor(t *testing.T) {
	t.Run("Process single batch", func(t *testing.T) {
		items := []int{1, 2, 3, 4, 5}
		bp := NewBatchProcessor[int](10)

		var processed []int
		err := bp.Process(items, func(batch []int) error {
			processed = append(processed, batch...)
			return nil
		})

		if err != nil {
			t.Errorf("batch processing failed: %v", err)
		}

		if len(processed) != len(items) {
			t.Errorf("expected %d processed items, got %d", len(items), len(processed))
		}
	})

	t.Run("Process multiple batches", func(t *testing.T) {
		items := make([]int, 25)
		for i := 0; i < 25; i++ {
			items[i] = i + 1
		}

		bp := NewBatchProcessor[int](10)
		batchCount := 0
		var processed []int

		err := bp.Process(items, func(batch []int) error {
			batchCount++
			processed = append(processed, batch...)
			return nil
		})

		if err != nil {
			t.Errorf("batch processing failed: %v", err)
		}

		if batchCount != 3 {
			t.Errorf("expected 3 batches, got %d", batchCount)
		}

		if len(processed) != 25 {
			t.Errorf("expected 25 processed items, got %d", len(processed))
		}
	})

	t.Run("Error in batch processing", func(t *testing.T) {
		items := []int{1, 2, 3, 4, 5}
		bp := NewBatchProcessor[int](2)

		err := bp.Process(items, func(batch []int) error {
			if batch[0] == 3 {
				return sql.ErrNoRows
			}
			return nil
		})

		if err == nil {
			t.Error("expected batch processing to fail")
		}
	})

	t.Run("Zero batch size defaults to 100", func(t *testing.T) {
		bp := NewBatchProcessor[int](0)
		if bp.batchSize != 100 {
			t.Errorf("expected default batch size 100, got %d", bp.batchSize)
		}

		bp = NewBatchProcessor[int](-5)
		if bp.batchSize != 100 {
			t.Errorf("expected default batch size 100 for negative input, got %d", bp.batchSize)
		}
	})
}

func TestNullHelpers(t *testing.T) {
	t.Run("NullString helpers", func(t *testing.T) {

		ns := ToNullString("test")
		if !ns.Valid || ns.String != "test" {
			t.Error("ToNullString failed for non-empty string")
		}

		ns = ToNullString("")
		if ns.Valid {
			t.Error("ToNullString should not be valid for empty string")
		}

		s := FromNullString(sql.NullString{String: "test", Valid: true})
		if s != "test" {
			t.Errorf("FromNullString failed, expected 'test', got '%s'", s)
		}

		s = FromNullString(sql.NullString{String: "test", Valid: false})
		if s != "" {
			t.Errorf("FromNullString should return empty string for invalid NullString, got '%s'", s)
		}
	})

	t.Run("NullInt64 helpers", func(t *testing.T) {

		ni := ToNullInt64(42)
		if !ni.Valid || ni.Int64 != 42 {
			t.Error("ToNullInt64 failed for non-zero int")
		}

		ni = ToNullInt64(0)
		if ni.Valid {
			t.Error("ToNullInt64 should not be valid for zero")
		}

		i := FromNullInt64(sql.NullInt64{Int64: 42, Valid: true})
		if i != 42 {
			t.Errorf("FromNullInt64 failed, expected 42, got %d", i)
		}

		i = FromNullInt64(sql.NullInt64{Int64: 42, Valid: false})
		if i != 0 {
			t.Errorf("FromNullInt64 should return 0 for invalid NullInt64, got %d", i)
		}
	})

	t.Run("NullTime helpers", func(t *testing.T) {
		now := time.Now()

		nt := ToNullTime(now)
		if !nt.Valid || !nt.Time.Equal(now) {
			t.Error("ToNullTime failed for non-zero time")
		}

		nt = ToNullTime(time.Time{})
		if nt.Valid {
			t.Error("ToNullTime should not be valid for zero time")
		}

		retrievedTime := FromNullTime(sql.NullTime{Time: now, Valid: true})
		if !retrievedTime.Equal(now) {
			t.Errorf("FromNullTime failed, expected %v, got %v", now, retrievedTime)
		}

		retrievedTime = FromNullTime(sql.NullTime{Time: now, Valid: false})
		if !retrievedTime.IsZero() {
			t.Error("FromNullTime should return zero time for invalid NullTime")
		}
	})
}

func TestValidateColumnName(t *testing.T) {
	tests := []struct {
		name        string
		columnName  string
		expectError bool
	}{
		{
			name:       "valid simple column",
			columnName: "name",
		},
		{
			name:       "valid column with underscore",
			columnName: "user_id",
		},
		{
			name:       "valid column with numbers",
			columnName: "column123",
		},
		{
			name:       "valid column starting with underscore",
			columnName: "_private",
		},
		{
			name:        "invalid column with space",
			columnName:  "user name",
			expectError: true,
		},
		{
			name:        "invalid column with semicolon",
			columnName:  "user;",
			expectError: true,
		},
		{
			name:        "invalid sql injection attempt",
			columnName:  "user; DROP TABLE users; --",
			expectError: true,
		},
		{
			name:        "invalid column starting with number",
			columnName:  "123column",
			expectError: true,
		},
		{
			name:        "invalid empty column",
			columnName:  "",
			expectError: true,
		},
		{
			name:        "invalid column with special characters",
			columnName:  "user@domain.com",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateColumnName(tt.columnName)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestConfigureConnectionPool(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	t.Run("Configure connection pool", func(t *testing.T) {
		ConfigureConnectionPool(db, 50, 10, 3600)

		err := db.Ping()
		if err != nil {
			t.Errorf("database should still be usable after pool configuration: %v", err)
		}
	})
}
