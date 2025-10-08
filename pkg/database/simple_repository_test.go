package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/shuldan/framework/pkg/contracts"
)

type TestUser struct {
	id    contracts.ID
	name  string
	email string
}

func (u TestUser) ID() contracts.ID {
	return u.id
}

func NewTestUser(id int64, name, email string) TestUser {
	return TestUser{
		id:    NewIntID(id),
		name:  name,
		email: email,
	}
}

type TestUserMemento struct {
	ID    contracts.ID
	Name  string
	Email string
}

func (m TestUserMemento) GetID() contracts.ID {
	return m.ID
}

type TestUserMapper struct{}

func (m *TestUserMapper) TableName() string {
	return "users"
}

func (m *TestUserMapper) IDColumn() string {
	return "id"
}

func (m *TestUserMapper) CreateMemento(aggregate TestUser) (TestUserMemento, error) {
	return TestUserMemento{
		ID:    aggregate.id,
		Name:  aggregate.name,
		Email: aggregate.email,
	}, nil
}

func (m *TestUserMapper) RestoreAggregate(memento TestUserMemento) (TestUser, error) {
	return TestUser{
		id:    memento.ID,
		name:  memento.Name,
		email: memento.Email,
	}, nil
}

func (m *TestUserMapper) GetColumns() []string {
	return []string{"id", "name", "email"}
}

func (m *TestUserMapper) GetValues(memento TestUserMemento) []interface{} {
	return []interface{}{memento.ID.String(), memento.Name, memento.Email}
}

func (m *TestUserMapper) FromRow(row *sql.Row) (TestUserMemento, error) {
	var memento TestUserMemento
	var id int64
	err := row.Scan(&id, &memento.Name, &memento.Email)
	if err != nil {
		return memento, err
	}
	memento.ID = NewIntID(id)
	return memento, nil
}

func (m *TestUserMapper) FromRows(rows *sql.Rows) (TestUserMemento, error) {
	var memento TestUserMemento
	var id int64
	err := rows.Scan(&id, &memento.Name, &memento.Email)
	if err != nil {
		return memento, err
	}
	memento.ID = NewIntID(id)
	return memento, nil
}

func TestSimpleRepository(t *testing.T) {
	db := setupTestDBWithUsers(t)

	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close database: %v", err)
		}
	}()

	mapper := &TestUserMapper{}
	repo := NewSimpleRepository[TestUser, contracts.ID, TestUserMemento](db, mapper)

	ctx := context.Background()

	testSaveAndFind(t, repo, ctx)
	testUpdate(t, repo, ctx)
	testFindAll(t, repo, ctx)
	testFindBy(t, repo, ctx)
	testCount(t, repo, ctx)
	testDelete(t, repo, ctx)
	testDeleteBy(t, repo, ctx)
	testInvalidOperations(t, repo, ctx)
}

func testSaveAndFind(t *testing.T, repo contracts.TransactionalRepository[TestUser, contracts.ID], ctx context.Context) {
	t.Run("Save new entity", func(t *testing.T) {
		user := NewTestUser(1, "John Doe", "john@example.com")
		err := repo.Save(ctx, user)
		if err != nil {
			t.Errorf("failed to save user: %v", err)
		}

		exists, err := repo.Exists(ctx, user.id)
		if err != nil {
			t.Errorf("failed to check if user exists: %v", err)
		}
		if !exists {
			t.Error("user should exist after save")
		}
	})

	t.Run("Find existing entity", func(t *testing.T) {
		id := NewIntID(1)
		user, err := repo.Find(ctx, id)
		if err != nil {
			t.Errorf("failed to find user: %v", err)
		}

		if user.id != id {
			t.Errorf("expected ID %v, got %v", id, user.id)
		}
		if user.name != "John Doe" {
			t.Errorf("expected name 'John Doe', got '%s'", user.name)
		}
		if user.email != "john@example.com" {
			t.Errorf("expected email 'john@example.com', got '%s'", user.email)
		}
	})

	t.Run("Find non-existing entity", func(t *testing.T) {
		id := NewIntID(999)
		_, err := repo.Find(ctx, id)
		if err == nil {
			t.Error("expected error when finding non-existing user")
		}
		if !errors.Is(err, ErrEntityNotFound) {
			t.Errorf("expected ErrEntityNotFound, got %v", err)
		}
	})
}

func testUpdate(t *testing.T, repo contracts.TransactionalRepository[TestUser, contracts.ID], ctx context.Context) {
	t.Run("Update existing entity", func(t *testing.T) {
		user := NewTestUser(1, "Jane Doe", "jane@example.com")
		err := repo.Save(ctx, user)
		if err != nil {
			t.Errorf("failed to update user: %v", err)
		}

		foundUser, err := repo.Find(ctx, user.id)
		if err != nil {
			t.Errorf("failed to find updated user: %v", err)
		}

		if foundUser.name != "Jane Doe" {
			t.Errorf("expected name 'Jane Doe', got '%s'", foundUser.name)
		}
		if foundUser.email != "jane@example.com" {
			t.Errorf("expected email 'jane@example.com', got '%s'", foundUser.email)
		}
	})
}

func testFindAll(t *testing.T, repo contracts.TransactionalRepository[TestUser, contracts.ID], ctx context.Context) {
	t.Run("FindAll", func(t *testing.T) {
		users := []TestUser{
			NewTestUser(2, "Alice", "alice@example.com"),
			NewTestUser(3, "Bob", "bob@example.com"),
		}

		for _, user := range users {
			err := repo.Save(ctx, user)
			if err != nil {
				t.Errorf("failed to save user %s: %v", user.name, err)
			}
		}

		allUsers, err := repo.FindAll(ctx, 10, 0)
		if err != nil {
			t.Errorf("failed to find all users: %v", err)
		}

		if len(allUsers) < 3 {
			t.Errorf("expected at least 3 users, got %d", len(allUsers))
		}
	})
}

func testFindBy(t *testing.T, repo contracts.TransactionalRepository[TestUser, contracts.ID], ctx context.Context) {
	t.Run("FindBy", func(t *testing.T) {
		users, err := repo.FindBy(ctx, map[string]interface{}{
			"name": "Jane Doe",
		})
		if err != nil {
			t.Errorf("failed to find users by name: %v", err)
		}

		if len(users) != 1 {
			t.Errorf("expected 1 user, got %d", len(users))
		}
		if len(users) > 0 && users[0].name != "Jane Doe" {
			t.Errorf("expected name 'Jane Doe', got '%s'", users[0].name)
		}
	})

	t.Run("FindBy with empty criteria", func(t *testing.T) {
		_, err := repo.FindBy(ctx, map[string]interface{}{})
		if err == nil {
			t.Error("expected error with empty criteria")
		}
	})

	t.Run("FindBy with invalid column name", func(t *testing.T) {
		_, err := repo.FindBy(ctx, map[string]interface{}{
			"invalid; DROP TABLE users; --": "value",
		})
		if err == nil {
			t.Error("expected error with invalid column name")
		}
	})
}

func testCount(t *testing.T, repo contracts.TransactionalRepository[TestUser, contracts.ID], ctx context.Context) {
	t.Run("Count", func(t *testing.T) {
		count, err := repo.Count(ctx, map[string]interface{}{})
		if err != nil {
			t.Errorf("failed to count all users: %v", err)
		}
		if count < 3 {
			t.Errorf("expected at least 3 users, got %d", count)
		}

		count, err = repo.Count(ctx, map[string]interface{}{
			"name": "Jane Doe",
		})
		if err != nil {
			t.Errorf("failed to count users by name: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 user named Jane Doe, got %d", count)
		}
	})
}

func testDelete(t *testing.T, repo contracts.TransactionalRepository[TestUser, contracts.ID], ctx context.Context) {
	t.Run("Delete", func(t *testing.T) {
		id := NewIntID(2)
		err := repo.Delete(ctx, id)
		if err != nil {
			t.Errorf("failed to delete user: %v", err)
		}

		exists, err := repo.Exists(ctx, id)
		if err != nil {
			t.Errorf("failed to check if user exists: %v", err)
		}
		if exists {
			t.Error("user should not exist after delete")
		}
	})

	t.Run("Delete non-existing entity", func(t *testing.T) {
		id := NewIntID(999)
		err := repo.Delete(ctx, id)
		if err == nil {
			t.Error("expected error when deleting non-existing user")
		}
		if !errors.Is(err, ErrEntityNotFound) {
			t.Errorf("expected ErrEntityNotFound, got %v", err)
		}
	})
}

func testDeleteBy(t *testing.T, repo contracts.TransactionalRepository[TestUser, contracts.ID], ctx context.Context) {
	t.Run("DeleteBy", func(t *testing.T) {
		user := NewTestUser(4, "Test User", "test@example.com")
		err := repo.Save(ctx, user)
		if err != nil {
			t.Errorf("failed to save user: %v", err)
		}

		count, err := repo.DeleteBy(ctx, map[string]interface{}{
			"name": "Test User",
		})
		if err != nil {
			t.Errorf("failed to delete users by name: %v", err)
		}
		if count != 1 {
			t.Errorf("expected to delete 1 user, got %d", count)
		}
	})

	t.Run("DeleteBy with empty criteria", func(t *testing.T) {
		_, err := repo.DeleteBy(ctx, map[string]interface{}{})
		if err == nil {
			t.Error("expected error with empty criteria")
		}
	})
}

func testInvalidOperations(t *testing.T, repo contracts.TransactionalRepository[TestUser, contracts.ID], ctx context.Context) {
	t.Run("Save with invalid ID", func(t *testing.T) {
		user := TestUser{
			id:    NewIntID(0),
			name:  "Invalid User",
			email: "invalid@example.com",
		}
		err := repo.Save(ctx, user)
		if err == nil {
			t.Error("expected error when saving user with invalid ID")
		}
	})
}

func TestSimpleRepositoryWithTransaction(t *testing.T) {
	database := setupTestDatabase(t)
	defer func() {
		if err := database.Close(); err != nil {
			t.Logf("failed to close database: %v", err)
		}
	}()

	sqlDB := database.(*sqlDatabase).db
	setupUsersTable(t, sqlDB)

	mapper := &TestUserMapper{}
	repo := NewSimpleRepository[TestUser, contracts.ID, TestUserMemento](sqlDB, mapper)

	ctx := context.Background()

	testSimpleRepoTransactionCommit(t, database, repo, ctx)
	testSimpleRepoTransactionRollback(t, database, repo, ctx)
}

func testSimpleRepoTransactionCommit(t *testing.T, database contracts.Database, repo contracts.TransactionalRepository[TestUser, contracts.ID], ctx context.Context) {
	t.Run("Transaction commit", func(t *testing.T) {
		tx, err := database.BeginTx(ctx)
		if err != nil {
			t.Fatalf("failed to begin transaction: %v", err)
		}

		txRepo := repo.WithTx(tx)
		user := NewTestUser(10, "Transaction User", "tx@example.com")

		err = txRepo.Save(ctx, user)
		if err != nil {
			t.Errorf("failed to save user in transaction: %v", err)
		}

		err = tx.Commit()
		if err != nil {
			t.Errorf("failed to commit transaction: %v", err)
		}

		exists, err := repo.Exists(ctx, user.id)
		if err != nil {
			t.Errorf("failed to check if user exists: %v", err)
		}
		if !exists {
			t.Error("user should exist after transaction commit")
		}
	})
}

func testSimpleRepoTransactionRollback(t *testing.T, database contracts.Database, repo contracts.TransactionalRepository[TestUser, contracts.ID], ctx context.Context) {
	t.Run("Transaction rollback", func(t *testing.T) {
		tx, err := database.BeginTx(ctx)
		if err != nil {
			t.Fatalf("failed to begin transaction: %v", err)
		}

		txRepo := repo.WithTx(tx)
		user := NewTestUser(11, "Rollback User", "rollback@example.com")

		err = txRepo.Save(ctx, user)
		if err != nil {
			t.Errorf("failed to save user in transaction: %v", err)
		}

		err = tx.Rollback()
		if err != nil {
			t.Errorf("failed to rollback transaction: %v", err)
		}

		exists, err := repo.Exists(ctx, user.id)
		if err != nil {
			t.Errorf("failed to check if user exists: %v", err)
		}
		if exists {
			t.Error("user should not exist after transaction rollback")
		}
	})
}

func setupTestDBWithUsers(t *testing.T) *sql.DB {
	db := setupTestDB(t)
	setupUsersTable(t, db)
	return db
}

func setupUsersTable(t *testing.T, db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}
}
