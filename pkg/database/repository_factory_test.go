package database

import (
	"context"
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
)

func TestRepositoryFactory(t *testing.T) {
	db := setupTestDBWithUsers(t)
	defer db.Close()

	t.Run("NewRepositoryFactory", func(t *testing.T) {
		factory := NewRepositoryFactory(db)
		if factory == nil {
			t.Fatal("NewRepositoryFactory returned nil")
		}

		if factory.GetDatabase() != db {
			t.Error("factory should return the same database instance")
		}
	})

	t.Run("CreateSimpleRepository", func(t *testing.T) {
		mapper := &TestUserMapper{}
		repo := CreateSimpleRepository[TestUser, IntID, TestUserMemento](db, mapper)

		if repo == nil {
			t.Fatal("CreateSimpleRepository returned nil")
		}

		var _ = repo

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

	t.Run("CreateStrategyRepository", func(t *testing.T) {
		mapper := &TestUserStrategyMapper{TestUserMapper: &TestUserMapper{}}
		repo := CreateStrategyRepository[TestUser, IntID, TestUserMemento](
			db, mapper, contracts.LoadingStrategyJoin)

		if repo == nil {
			t.Fatal("CreateStrategyRepository returned nil")
		}

		var _ = repo

		if repo.GetStrategy() != contracts.LoadingStrategyJoin {
			t.Errorf("expected LoadingStrategyJoin, got %s", repo.GetStrategy())
		}

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

		multipleRepo := repo.WithStrategy(contracts.LoadingStrategyMultiple)
		foundUser2, err := multipleRepo.Find(ctx, user.id)
		if err != nil {
			t.Errorf("failed to find user with different strategy: %v", err)
		}

		if foundUser2.name != user.name {
			t.Errorf("expected name '%s', got '%s'", user.name, foundUser2.name)
		}
	})
}
