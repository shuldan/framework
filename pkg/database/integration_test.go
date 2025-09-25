package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

func TestDatabaseIntegration(t *testing.T) {
	database := setupTestDatabase(t)
	defer closeDatabaseSafely(t, database)

	ctx := context.Background()

	t.Run("Complete workflow", func(t *testing.T) {
		testCompleteWorkflow(t, database, ctx)
	})

	t.Run("Transaction workflow", func(t *testing.T) {
		testTransactionWorkflow(t, database, ctx)
	})

	t.Run("Migration rollback workflow", func(t *testing.T) {
		testMigrationRollbackWorkflow(t, database)
	})
}

func testCompleteWorkflow(t *testing.T, database contracts.Database, ctx context.Context) {
	migration := CreateMigration("001", "create users table").
		CreateTable("users",
			"id INTEGER PRIMARY KEY",
			"name TEXT NOT NULL",
			"email TEXT NOT NULL").
		CreateIndex("idx_users_email", "users", "email").
		Build()

	err := database.Migrate([]contracts.Migration{migration})
	if err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	sqlDB := database.(*sqlDatabase).db
	mapper := &TestUserMapper{}
	repo := NewSimpleRepository[TestUser, contracts.ID, TestUserMemento](sqlDB, mapper)

	testUserOperations(t, repo, ctx)
	testIntegrationMigrationStatus(t, database)
}

func testUserOperations(t *testing.T, repo contracts.TransactionalRepository[TestUser, contracts.ID], ctx context.Context) {
	user := NewTestUser(1, "Integration User", "integration@example.com")

	err := repo.Save(ctx, user)
	if err != nil {
		t.Errorf("failed to save user: %v", err)
	}

	foundUser, err := repo.Find(ctx, user.id)
	if err != nil {
		t.Errorf("failed to find user: %v", err)
	}
	if foundUser.name != user.name {
		t.Errorf("expected name '%s', got '%s'", user.name, foundUser.name)
	}

	updatedUser := NewTestUser(1, "Updated User", "updated@example.com")
	err = repo.Save(ctx, updatedUser)
	if err != nil {
		t.Errorf("failed to update user: %v", err)
	}

	foundUser, err = repo.Find(ctx, user.id)
	if err != nil {
		t.Errorf("failed to find updated user: %v", err)
	}
	if foundUser.name != "Updated User" {
		t.Errorf("expected name 'Updated User', got '%s'", foundUser.name)
	}

	err = repo.Delete(ctx, user.id)
	if err != nil {
		t.Errorf("failed to delete user: %v", err)
	}

	_, err = repo.Find(ctx, user.id)
	if err == nil {
		t.Error("expected error when finding deleted user")
	}
}

func testIntegrationMigrationStatus(t *testing.T, database contracts.Database) {
	status, err := database.Status()
	if err != nil {
		t.Errorf("failed to get migration status: %v", err)
	}

	if len(status) != 1 {
		t.Errorf("expected 1 migration, got %d", len(status))
	}
	if status[0].ID != "001" {
		t.Errorf("expected migration ID '001', got '%s'", status[0].ID)
	}
}

func testTransactionWorkflow(t *testing.T, database contracts.Database, ctx context.Context) {
	sqlDB := database.(*sqlDatabase).db
	mapper := &TestUserMapper{}
	repo := NewSimpleRepository[TestUser, contracts.ID, TestUserMemento](sqlDB, mapper)

	testTransactionCommit(t, database, repo, ctx)
	testTransactionRollback(t, database, repo, ctx)
}

func testTransactionCommit(t *testing.T, database contracts.Database, repo contracts.TransactionalRepository[TestUser, contracts.ID], ctx context.Context) {
	tx, err := database.BeginTx(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	txRepo := repo.WithTx(tx)

	user1 := NewTestUser(10, "TX User 1", "tx1@example.com")
	user2 := NewTestUser(11, "TX User 2", "tx2@example.com")

	err = txRepo.Save(ctx, user1)
	if err != nil {
		t.Errorf("failed to save user1 in transaction: %v", err)
	}

	err = txRepo.Save(ctx, user2)
	if err != nil {
		t.Errorf("failed to save user2 in transaction: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Errorf("failed to commit transaction: %v", err)
	}

	for _, user := range []TestUser{user1, user2} {
		exists, err := repo.Exists(ctx, user.id)
		if err != nil {
			t.Errorf("failed to check existence of %s: %v", user.name, err)
		}
		if !exists {
			t.Errorf("user %s should exist after commit", user.name)
		}
	}
}

func testTransactionRollback(t *testing.T, database contracts.Database, repo contracts.TransactionalRepository[TestUser, contracts.ID], ctx context.Context) {
	tx, err := database.BeginTx(ctx)
	if err != nil {
		t.Fatalf("failed to begin rollback transaction: %v", err)
	}

	txRepo := repo.WithTx(tx)
	user3 := NewTestUser(12, "TX User 3", "tx3@example.com")

	err = txRepo.Save(ctx, user3)
	if err != nil {
		t.Errorf("failed to save user3 in transaction: %v", err)
	}

	err = tx.Rollback()
	if err != nil {
		t.Errorf("failed to rollback transaction: %v", err)
	}

	exists, err := repo.Exists(ctx, user3.id)
	if err != nil {
		t.Errorf("failed to check existence of user3: %v", err)
	}
	if exists {
		t.Error("user3 should not exist after rollback")
	}
}

func testMigrationRollbackWorkflow(t *testing.T, database contracts.Database) {
	migration2 := CreateMigration("002", "create posts table").
		CreateTable("posts",
			"id INTEGER PRIMARY KEY",
			"title TEXT NOT NULL",
			"user_id INTEGER",
			"FOREIGN KEY(user_id) REFERENCES users(id)").
		Build()

	migration3 := CreateMigration("003", "add posts index").
		CreateIndex("idx_posts_user_id", "posts", "user_id").
		Build()

	migrations := []contracts.Migration{migration2, migration3}
	err := database.Migrate(migrations)
	if err != nil {
		t.Errorf("failed to run additional migrations: %v", err)
	}

	status, err := database.Status()
	if err != nil {
		t.Errorf("failed to get migration status: %v", err)
	}

	if len(status) < 3 {
		t.Errorf("expected at least 3 migrations, got %d", len(status))
	}

	err = database.Rollback(1, migrations)
	if err != nil {
		t.Errorf("failed to rollback migration: %v", err)
	}

	status, err = database.Status()
	if err != nil {
		t.Errorf("failed to get migration status after rollback: %v", err)
	}

	found003 := false
	for _, s := range status {
		if s.ID == "003" {
			found003 = true
			break
		}
	}
	if found003 {
		t.Error("migration 003 should have been rolled back")
	}
}

func TestDatabaseConnectionPooling(t *testing.T) {
	t.Run("Connection pool configuration", func(t *testing.T) {
		database := NewDatabase("sqlite3", ":memory:",
			WithConnectionPool(10, 5, time.Hour),
			WithPingTimeout(time.Second*10),
			WithRetry(3, time.Millisecond*100))

		err := database.Connect()
		if err != nil {
			t.Errorf("failed to connect with pool configuration: %v", err)
		}

		defer closeDatabaseSafely(t, database)

		ctx := context.Background()
		err = database.Ping(ctx)
		if err != nil {
			t.Errorf("ping failed with pool configuration: %v", err)
		}
	})
}

func TestDatabaseErrorScenarios(t *testing.T) {
	testBadMigration(t)
	testNonExistentTable(t)
	testConnectionRetry(t)
}

func testBadMigration(t *testing.T) {
	t.Run("Migration with syntax error", func(t *testing.T) {
		database := setupTestDatabase(t)
		defer closeDatabaseSafely(t, database)

		badMigration := CreateMigration("bad001", "bad migration").
			RawUp("INVALID SQL SYNTAX HERE").
			Build()

		err := database.Migrate([]contracts.Migration{badMigration})
		if err == nil {
			t.Error("expected error for bad migration")
		}
	})
}

func testNonExistentTable(t *testing.T) {
	t.Run("Repository operations on non-existent table", func(t *testing.T) {
		database := setupTestDatabase(t)
		defer closeDatabaseSafely(t, database)

		sqlDB := database.(*sqlDatabase).db
		mapper := &TestUserMapper{}
		repo := NewSimpleRepository[TestUser, contracts.ID, TestUserMemento](sqlDB, mapper)

		ctx := context.Background()
		user := NewTestUser(1, "Test User", "test@example.com")

		err := repo.Save(ctx, user)
		if err == nil {
			t.Error("expected error when saving to non-existent table")
		}
	})
}

func testConnectionRetry(t *testing.T) {
	t.Run("Connection retry on failure", func(t *testing.T) {
		database := NewDatabase("sqlite3", "/invalid/path/database.db",
			WithRetry(2, time.Millisecond*10))

		start := time.Now()
		err := database.Connect()
		duration := time.Since(start)

		if err == nil {
			t.Error("expected connection to fail")
		}

		expectedMinDuration := time.Millisecond * 20
		if duration < expectedMinDuration {
			t.Errorf("expected at least %v duration for retries, got %v", expectedMinDuration, duration)
		}
	})
}

func TestConcurrentDatabaseOperations(t *testing.T) {
	database := setupTestDatabase(t)
	defer closeDatabaseSafely(t, database)

	migration := CreateMigration("001", "create users").
		CreateTable("users", "id INTEGER PRIMARY KEY", "name TEXT", "email TEXT").
		Build()

	err := database.Migrate([]contracts.Migration{migration})
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	sqlDB := database.(*sqlDatabase).db
	mapper := &TestUserMapper{}
	repo := NewSimpleRepository[TestUser, contracts.ID, TestUserMemento](sqlDB, mapper)

	testConcurrentSaves(t, repo)
	testConcurrentTransactions(t, database, repo)
}

func testConcurrentSaves(t *testing.T, repo contracts.TransactionalRepository[TestUser, contracts.ID]) {
	t.Run("Concurrent saves", func(t *testing.T) {
		const numGoroutines = 10
		errChan := make(chan error, numGoroutines)
		ctx := context.Background()

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				user := NewTestUser(int64(100+id),
					fmt.Sprintf("Concurrent User %d", id),
					fmt.Sprintf("concurrent%d@example.com", id))
				errChan <- repo.Save(ctx, user)
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			if err := <-errChan; err != nil {
				t.Errorf("concurrent save failed: %v", err)
			}
		}

		count, err := repo.Count(ctx, map[string]interface{}{})
		if err != nil {
			t.Errorf("failed to count users: %v", err)
		}
		if count < int64(numGoroutines) {
			t.Errorf("expected at least %d users, got %d", numGoroutines, count)
		}
	})
}

func testConcurrentTransactions(t *testing.T, database contracts.Database, repo contracts.TransactionalRepository[TestUser, contracts.ID]) {
	t.Run("Concurrent transactions", func(t *testing.T) {
		const numGoroutines = 5
		errChan := make(chan error, numGoroutines)
		ctx := context.Background()

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				tx, err := database.BeginTx(ctx)
				if err != nil {
					errChan <- err
					return
				}

				txRepo := repo.WithTx(tx)
				user := NewTestUser(int64(200+id),
					fmt.Sprintf("TX User %d", id),
					fmt.Sprintf("tx%d@example.com", id))

				err = txRepo.Save(ctx, user)
				if err != nil {
					if err := tx.Rollback(); err != nil {
						t.Errorf("failed to rollback transaction: %v", err)
					}
					errChan <- err
					return
				}

				errChan <- tx.Commit()
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			if err := <-errChan; err != nil {
				t.Errorf("concurrent transaction failed: %v", err)
			}
		}
	})
}

func closeDatabaseSafely(t *testing.T, database contracts.Database) {
	if err := database.Close(); err != nil {
		t.Logf("failed to close database: %v", err)
	}
}
