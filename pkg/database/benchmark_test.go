package database

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
)

func BenchmarkSimpleRepository(b *testing.B) {
	database := setupTestDatabaseForBenchmark(b)
	defer database.Close()

	migration := CreateMigration("001", "create users").
		CreateTable("users", "id INTEGER PRIMARY KEY", "name TEXT", "email TEXT").
		Build()

	err := database.Migrate([]contracts.Migration{migration})
	if err != nil {
		b.Fatalf("failed to create table: %v", err)
	}

	sqlDB := database.(*sqlDatabase).db
	mapper := &TestUserMapper{}
	repo := NewSimpleRepository[TestUser, IntID, TestUserMemento](sqlDB, mapper)
	ctx := context.Background()

	b.Run("Save", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			user := NewTestUser(int64(i+1), fmt.Sprintf("User %d", i), fmt.Sprintf("user%d@example.com", i))
			err := repo.Save(ctx, user)
			if err != nil {
				b.Errorf("save failed: %v", err)
			}
		}
	})

	setupUsers := make([]TestUser, 1000)
	for i := 0; i < 1000; i++ {
		user := NewTestUser(int64(i+10000), fmt.Sprintf("Setup User %d", i), fmt.Sprintf("setup%d@example.com", i))
		setupUsers[i] = user
		repo.Save(ctx, user)
	}

	b.Run("Find", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			idx := i % len(setupUsers)
			_, err := repo.Find(ctx, setupUsers[idx].id)
			if err != nil {
				b.Errorf("find failed: %v", err)
			}
		}
	})

	b.Run("FindAll", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := repo.FindAll(ctx, 100, 0)
			if err != nil {
				b.Errorf("findall failed: %v", err)
			}
		}
	})

	b.Run("Exists", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			idx := i % len(setupUsers)
			_, err := repo.Exists(ctx, setupUsers[idx].id)
			if err != nil {
				b.Errorf("exists failed: %v", err)
			}
		}
	})

	b.Run("Count", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := repo.Count(ctx, map[string]interface{}{})
			if err != nil {
				b.Errorf("count failed: %v", err)
			}
		}
	})
}

func BenchmarkStrategyRepository(b *testing.B) {
	database := setupTestDatabaseForBenchmark(b)
	defer database.Close()

	migration := CreateMigration("001", "create users").
		CreateTable("users", "id INTEGER PRIMARY KEY", "name TEXT", "email TEXT").
		Build()

	err := database.Migrate([]contracts.Migration{migration})
	if err != nil {
		b.Fatalf("failed to create table: %v", err)
	}

	sqlDB := database.(*sqlDatabase).db
	mapper := &TestUserStrategyMapper{TestUserMapper: &TestUserMapper{}}
	ctx := context.Background()

	setupUsers := make([]TestUser, 100)
	for i := 0; i < 100; i++ {
		user := NewTestUser(int64(i+1), fmt.Sprintf("User %d", i), fmt.Sprintf("user%d@example.com", i))
		setupUsers[i] = user
	}

	strategies := []contracts.LoadingStrategy{
		contracts.LoadingStrategyMultiple,
		contracts.LoadingStrategyJoin,
		contracts.LoadingStrategyBatch,
	}

	for _, strategy := range strategies {
		repo := NewStrategyRepository[TestUser, IntID, TestUserMemento](sqlDB, mapper, strategy)

		for _, user := range setupUsers {
			repo.Save(ctx, user)
		}

		b.Run(fmt.Sprintf("Find_%s", strategy), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				idx := i % len(setupUsers)
				_, err := repo.Find(ctx, setupUsers[idx].id)
				if err != nil {
					b.Errorf("find failed: %v", err)
				}
			}
		})

		b.Run(fmt.Sprintf("FindAll_%s", strategy), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := repo.FindAll(ctx, 50, 0)
				if err != nil {
					b.Errorf("findall failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkMigrationRunner(b *testing.B) {
	b.Run("Migration execution", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			database := setupTestDatabaseForBenchmark(b)
			runner := database.GetMigrationRunner()

			migration := CreateMigration(fmt.Sprintf("bench_%d", i), "benchmark migration").
				CreateTable(fmt.Sprintf("bench_table_%d", i), "id INTEGER PRIMARY KEY", "data TEXT").
				Build()

			err := runner.Run([]contracts.Migration{migration})
			if err != nil {
				b.Errorf("migration failed: %v", err)
			}

			database.Close()
		}
	})
}

func BenchmarkQueryBuilder(b *testing.B) {
	b.Run("Simple query building", func(b *testing.B) {
		qb := NewQueryBuilder()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			qb.Reset()
			qb.Select("*").From("users").Where("id = ?", i).Build()
		}
	})

	b.Run("Complex query building", func(b *testing.B) {
		qb := NewQueryBuilder()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			qb.Reset()
			qb.Select("u.name", "p.title").
				From("users u").
				Join("posts p", "p.user_id = u.id").
				Where("u.active = ?", true).
				And("p.published = ?", true).
				OrderBy("u.name", "ASC").
				Limit(10).
				Offset(i * 10).
				Build()
		}
	})
}

func BenchmarkTransactionManager(b *testing.B) {
	database := setupTestDatabaseForBenchmark(b)
	defer database.Close()

	sqlDB := database.(*sqlDatabase).db
	tm := NewTransactionManager(sqlDB)

	sqlDB.Exec("CREATE TABLE bench_tx (id INTEGER PRIMARY KEY, value TEXT)")

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := tm.Execute(ctx, func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, "INSERT INTO bench_tx (value) VALUES (?)", fmt.Sprintf("value_%d", i))
			return err
		})
		if err != nil {
			b.Errorf("transaction failed: %v", err)
		}
	}
}

func setupTestDatabaseForBenchmark(tb testing.TB) contracts.Database {
	database := NewDatabase("sqlite3", ":memory:")
	if err := database.Connect(); err != nil {
		tb.Fatalf("failed to connect to test database: %v", err)
	}
	return database
}
