package database

import (
	"context"
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
)

type TestUserStrategyMapper struct {
	*TestUserMapper
}

func (m *TestUserStrategyMapper) FindByIDMultiple(ctx context.Context, db QueryExecutor, id contracts.ID) (TestUserMemento, error) {
	query := "SELECT id, name, email FROM users WHERE id = ?"
	row := db.QueryRowContext(ctx, query, id.String())
	return m.FromRow(row)
}

func (m *TestUserStrategyMapper) FindByIDJoin(ctx context.Context, db QueryExecutor, id contracts.ID) (TestUserMemento, error) {
	return m.FindByIDMultiple(ctx, db, id)
}

func (m *TestUserStrategyMapper) FindByIDBatch(ctx context.Context, db QueryExecutor, ids []contracts.ID) ([]TestUserMemento, error) {
	if len(ids) == 0 {
		return []TestUserMemento{}, nil
	}

	query := "SELECT id, name, email FROM users WHERE id IN (?"
	args := []interface{}{ids[0].String()}
	for i := 1; i < len(ids); i++ {
		query += ",?"
		args = append(args, ids[i].String())
	}
	query += ")"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mementos []TestUserMemento
	for rows.Next() {
		memento, err := m.FromRows(rows)
		if err != nil {
			return nil, err
		}
		mementos = append(mementos, memento)
	}

	return mementos, rows.Err()
}

func (m *TestUserStrategyMapper) FindAllMultiple(ctx context.Context, db QueryExecutor, limit, offset int) ([]TestUserMemento, error) {
	query := "SELECT id, name, email FROM users LIMIT ? OFFSET ?"
	rows, err := db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mementos []TestUserMemento
	for rows.Next() {
		memento, err := m.FromRows(rows)
		if err != nil {
			return nil, err
		}
		mementos = append(mementos, memento)
	}

	return mementos, rows.Err()
}

func (m *TestUserStrategyMapper) FindAllJoin(ctx context.Context, db QueryExecutor, limit, offset int) ([]TestUserMemento, error) {
	return m.FindAllMultiple(ctx, db, limit, offset)
}

func (m *TestUserStrategyMapper) FindAllBatch(ctx context.Context, db QueryExecutor, limit, offset int) ([]TestUserMemento, error) {
	return m.FindAllMultiple(ctx, db, limit, offset)
}

func (m *TestUserStrategyMapper) FindByMultiple(ctx context.Context, db QueryExecutor, criteria map[string]interface{}) ([]TestUserMemento, error) {
	query := "SELECT id, name, email FROM users WHERE "
	var conditions []string
	var args []interface{}

	for field, value := range criteria {
		conditions = append(conditions, field+" = ?")
		args = append(args, value)
	}

	query += conditions[0]
	for i := 1; i < len(conditions); i++ {
		query += " AND " + conditions[i]
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mementos []TestUserMemento
	for rows.Next() {
		memento, err := m.FromRows(rows)
		if err != nil {
			return nil, err
		}
		mementos = append(mementos, memento)
	}

	return mementos, rows.Err()
}

func (m *TestUserStrategyMapper) FindByJoin(ctx context.Context, db QueryExecutor, criteria map[string]interface{}) ([]TestUserMemento, error) {
	return m.FindByMultiple(ctx, db, criteria)
}

func (m *TestUserStrategyMapper) FindByBatch(ctx context.Context, db QueryExecutor, criteria map[string]interface{}) ([]TestUserMemento, error) {
	return m.FindByMultiple(ctx, db, criteria)
}

func (m *TestUserStrategyMapper) SaveWithRelations(ctx context.Context, db QueryExecutor, memento TestUserMemento, isUpdate bool) error {
	if isUpdate {
		query := "UPDATE users SET name = ?, email = ? WHERE id = ?"
		_, err := db.ExecContext(ctx, query, memento.Name, memento.Email, memento.ID.Int64())
		return err
	} else {
		query := "INSERT INTO users (id, name, email) VALUES (?, ?, ?)"
		_, err := db.ExecContext(ctx, query, memento.ID.Int64(), memento.Name, memento.Email)
		return err
	}
}

func (m *TestUserStrategyMapper) DeleteWithRelations(ctx context.Context, db QueryExecutor, id contracts.ID) error {
	query := "DELETE FROM users WHERE id = ?"
	_, err := db.ExecContext(ctx, query, id.String())
	return err
}

func TestStrategyRepository(t *testing.T) {
	db := setupTestDBWithUsers(t)
	defer db.Close()

	mapper := &TestUserStrategyMapper{TestUserMapper: &TestUserMapper{}}
	repo := NewStrategyRepository[TestUser, IntID, TestUserMemento](
		db, mapper, contracts.LoadingStrategyMultiple)

	ctx := context.Background()

	t.Run("Default strategy", func(t *testing.T) {
		strategy := repo.GetStrategy()
		if strategy != contracts.LoadingStrategyMultiple {
			t.Errorf("expected LoadingStrategyMultiple, got %s", strategy)
		}
	})

	t.Run("WithStrategy", func(t *testing.T) {
		joinRepo := repo.WithStrategy(contracts.LoadingStrategyJoin)

		if repo.GetStrategy() != contracts.LoadingStrategyMultiple {
			t.Error("original repo strategy should not change")
		}

		user := NewTestUser(1, "Strategy Test", "strategy@example.com")
		err := joinRepo.Save(ctx, user)
		if err != nil {
			t.Errorf("failed to save with join strategy: %v", err)
		}
	})

	t.Run("Find with different strategies", func(t *testing.T) {

		user := NewTestUser(2, "Multi Strategy", "multi@example.com")
		err := repo.Save(ctx, user)
		if err != nil {
			t.Errorf("failed to save user: %v", err)
		}

		strategies := []contracts.LoadingStrategy{
			contracts.LoadingStrategyMultiple,
			contracts.LoadingStrategyJoin,
			contracts.LoadingStrategyBatch,
		}

		for _, strategy := range strategies {
			t.Run(string(strategy), func(t *testing.T) {
				strategyRepo := repo.WithStrategy(strategy)
				foundUser, err := strategyRepo.Find(ctx, user.id)
				if err != nil {
					t.Errorf("failed to find user with %s strategy: %v", strategy, err)
				}

				if foundUser.name != user.name {
					t.Errorf("expected name '%s', got '%s'", user.name, foundUser.name)
				}
			})
		}
	})

	t.Run("FindAll with different strategies", func(t *testing.T) {

		users := []TestUser{
			NewTestUser(3, "User A", "a@example.com"),
			NewTestUser(4, "User B", "b@example.com"),
		}

		for _, user := range users {
			err := repo.Save(ctx, user)
			if err != nil {
				t.Errorf("failed to save user: %v", err)
			}
		}

		strategies := []contracts.LoadingStrategy{
			contracts.LoadingStrategyMultiple,
			contracts.LoadingStrategyJoin,
			contracts.LoadingStrategyBatch,
		}

		for _, strategy := range strategies {
			t.Run("FindAll_"+string(strategy), func(t *testing.T) {
				strategyRepo := repo.WithStrategy(strategy)
				foundUsers, err := strategyRepo.FindAll(ctx, 10, 0)
				if err != nil {
					t.Errorf("failed to find all users with %s strategy: %v", strategy, err)
				}

				if len(foundUsers) < 2 {
					t.Errorf("expected at least 2 users, got %d", len(foundUsers))
				}
			})
		}
	})

	t.Run("FindBy with different strategies", func(t *testing.T) {
		strategies := []contracts.LoadingStrategy{
			contracts.LoadingStrategyMultiple,
			contracts.LoadingStrategyJoin,
			contracts.LoadingStrategyBatch,
		}

		for _, strategy := range strategies {
			t.Run("FindBy_"+string(strategy), func(t *testing.T) {
				strategyRepo := repo.WithStrategy(strategy)
				foundUsers, err := strategyRepo.FindBy(ctx, map[string]interface{}{
					"name": "User A",
				})
				if err != nil {
					t.Errorf("failed to find users with %s strategy: %v", strategy, err)
				}

				if len(foundUsers) != 1 {
					t.Errorf("expected 1 user, got %d", len(foundUsers))
				}
				if len(foundUsers) > 0 && foundUsers[0].name != "User A" {
					t.Errorf("expected name 'User A', got '%s'", foundUsers[0].name)
				}
			})
		}
	})

	t.Run("Save and Delete with relations", func(t *testing.T) {
		user := NewTestUser(5, "Relations Test", "relations@example.com")

		err := repo.Save(ctx, user)
		if err != nil {
			t.Errorf("failed to save user: %v", err)
		}

		exists, err := repo.Exists(ctx, user.id)
		if err != nil {
			t.Errorf("failed to check existence: %v", err)
		}
		if !exists {
			t.Error("user should exist after save")
		}

		err = repo.Delete(ctx, user.id)
		if err != nil {
			t.Errorf("failed to delete user: %v", err)
		}

		exists, err = repo.Exists(ctx, user.id)
		if err != nil {
			t.Errorf("failed to check existence after delete: %v", err)
		}
		if exists {
			t.Error("user should not exist after delete")
		}
	})

	t.Run("Transaction support", func(t *testing.T) {
		database := setupTestDatabase(t)
		defer database.Close()

		sqlDB := database.(*sqlDatabase).db
		setupUsersTable(t, sqlDB)

		strategyRepo := NewStrategyRepository[TestUser, IntID, TestUserMemento](
			sqlDB, mapper, contracts.LoadingStrategyMultiple)

		tx, err := database.BeginTx(ctx)
		if err != nil {
			t.Fatalf("failed to begin transaction: %v", err)
		}

		txRepo := strategyRepo.WithTx(tx)
		user := NewTestUser(6, "TX Test", "tx@example.com")

		err = txRepo.Save(ctx, user)
		if err != nil {
			t.Errorf("failed to save in transaction: %v", err)
		}

		err = tx.Commit()
		if err != nil {
			t.Errorf("failed to commit: %v", err)
		}

		exists, err := strategyRepo.Exists(ctx, user.id)
		if err != nil {
			t.Errorf("failed to check existence: %v", err)
		}
		if !exists {
			t.Error("user should exist after transaction commit")
		}
	})

	t.Run("Invalid ID handling", func(t *testing.T) {
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

	t.Run("Error handling for non-existing entity", func(t *testing.T) {
		nonExistingID := NewIntID(999)

		_, err := repo.Find(ctx, nonExistingID)
		if err == nil {
			t.Error("expected error when finding non-existing user")
		}

		err = repo.Delete(ctx, nonExistingID)
		if err == nil {
			t.Error("expected error when deleting non-existing user")
		}
	})
}

func TestStrategyRepositoryBatchOperations(t *testing.T) {
	db := setupTestDBWithUsers(t)
	defer db.Close()

	mapper := &TestUserStrategyMapper{TestUserMapper: &TestUserMapper{}}
	repo := NewStrategyRepository[TestUser, IntID, TestUserMemento](
		db, mapper, contracts.LoadingStrategyBatch)

	ctx := context.Background()

	t.Run("Batch find by IDs", func(t *testing.T) {

		users := []TestUser{
			NewTestUser(10, "Batch User 1", "batch1@example.com"),
			NewTestUser(11, "Batch User 2", "batch2@example.com"),
			NewTestUser(12, "Batch User 3", "batch3@example.com"),
		}

		for _, user := range users {
			err := repo.Save(ctx, user)
			if err != nil {
				t.Errorf("failed to save user: %v", err)
			}
		}

		foundUser, err := repo.Find(ctx, users[0].id)
		if err != nil {
			t.Errorf("failed to find user: %v", err)
		}
		if foundUser.name != users[0].name {
			t.Errorf("expected name '%s', got '%s'", users[0].name, foundUser.name)
		}
	})

	t.Run("Batch find with empty result", func(t *testing.T) {

		_, err := repo.Find(ctx, NewIntID(999))
		if err == nil {
			t.Error("expected error when batch returns empty results")
		}
	})
}
