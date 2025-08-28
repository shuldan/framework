package database

import (
	"context"
	"database/sql"
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
)

func TestRepositoryFactory(t *testing.T) {
	db := setupTestDBWithUsers(t)

	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close database: %v", err)
		}
	}()

	testNewRepositoryFactory(t, db)
	testCreateSimpleRepository(t, db)
	testCreateStrategyRepository(t, db)
}

func testNewRepositoryFactory(t *testing.T, db *sql.DB) {
	t.Run("NewRepositoryFactory", func(t *testing.T) {
		factory := NewRepositoryFactory(db)
		if factory == nil {
			t.Fatal("NewRepositoryFactory returned nil")
		}

		if factory.GetDatabase() != db {
			t.Error("factory should return the same database instance")
		}
	})
}

func testCreateSimpleRepository(t *testing.T, db *sql.DB) {
	t.Run("CreateSimpleRepository", func(t *testing.T) {
		mapper := &TestUserMapper{}
		repo := CreateSimpleRepository[TestUser, IntID, TestUserMemento](db, mapper)

		if repo == nil {
			t.Fatal("CreateSimpleRepository returned nil")
		}

		ctx := context.Background()
		user := NewTestUser(1, "Factory User", "factory@example.com")

		err := repo.Save(ctx, user)
		if err != nil {
			t.Errorf("failed to save user through factory-created repo: %v", err)
		}

		foundUser, err := repo.Find(ctx, user.id)
		if err != nil {
			t.Errorf("failed to find user through factory-created repo: %v", err)
		}

		if foundUser.name != user.name {
			t.Errorf("expected name '%s', got '%s'", user.name, foundUser.name)
		}
	})
}

func testCreateStrategyRepository(t *testing.T, db *sql.DB) {
	t.Run("CreateStrategyRepository", func(t *testing.T) {
		mapper := &TestUserStrategyMapper{TestUserMapper: &TestUserMapper{}}
		repo := CreateStrategyRepository[TestUser, IntID, TestUserMemento](
			db, mapper, contracts.LoadingStrategyJoin)

		if repo == nil {
			t.Fatal("CreateStrategyRepository returned nil")
		}

		if repo.GetStrategy() != contracts.LoadingStrategyJoin {
			t.Errorf("expected LoadingStrategyJoin, got %s", repo.GetStrategy())
		}

		testStrategyRepositoryOperations(t, repo)
		testStrategyRepositoryWithDifferentStrategy(t, repo)
	})
}

func testStrategyRepositoryOperations(t *testing.T, repo contracts.StrategyRepository[TestUser, IntID]) {
	ctx := context.Background()
	user := NewTestUser(2, "Strategy Factory User", "strategy_factory@example.com")

	err := repo.Save(ctx, user)
	if err != nil {
		t.Errorf("failed to save user through factory-created strategy repo: %v", err)
	}

	foundUser, err := repo.Find(ctx, user.id)
	if err != nil {
		t.Errorf("failed to find user through factory-created strategy repo: %v", err)
	}

	if foundUser.name != user.name {
		t.Errorf("expected name '%s', got '%s'", user.name, foundUser.name)
	}
}

func testStrategyRepositoryWithDifferentStrategy(t *testing.T, repo contracts.StrategyRepository[TestUser, IntID]) {
	ctx := context.Background()
	user := NewTestUser(2, "Strategy Factory User", "strategy_factory@example.com")

	multipleRepo := repo.WithStrategy(contracts.LoadingStrategyMultiple)
	foundUser2, err := multipleRepo.Find(ctx, user.id)
	if err != nil {
		t.Errorf("failed to find user with different strategy: %v", err)
	}

	if foundUser2.name != user.name {
		t.Errorf("expected name '%s', got '%s'", user.name, foundUser2.name)
	}
}
