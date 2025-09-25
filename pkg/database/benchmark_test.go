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
	defer closeDatabaseWithLog(b, database)

	if err := createTestTable(database); err != nil {
		b.Fatalf("failed to create table: %v", err)
	}

	sqlDB := database.(*sqlDatabase).db
	mapper := &TestUserMapper{}
	repo := NewSimpleRepository[TestUser, contracts.ID, TestUserMemento](sqlDB, mapper)
	ctx := context.Background()

	benchmarkSave(b, repo, ctx)
	setupUsers := setupBenchmarkUsers(b, repo, ctx)
	benchmarkFind(b, repo, ctx, setupUsers)
	benchmarkFindAll(b, repo, ctx)
	benchmarkExists(b, repo, ctx, setupUsers)
	benchmarkCount(b, repo, ctx)
}

func benchmarkSave(b *testing.B, repo contracts.TransactionalRepository[TestUser, contracts.ID], ctx context.Context) {
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
}

func setupBenchmarkUsers(b *testing.B, repo contracts.TransactionalRepository[TestUser, contracts.ID], ctx context.Context) []TestUser {
	setupUsers := make([]TestUser, 1000)
	for i := 0; i < 1000; i++ {
		user := NewTestUser(int64(i+10000), fmt.Sprintf("Setup User %d", i), fmt.Sprintf("setup%d@example.com", i))
		setupUsers[i] = user
		err := repo.Save(ctx, user)
		if err != nil {
			b.Skipf("failed to setup user #%d: %v", i, err)
		}
	}
	return setupUsers
}

func benchmarkFind(b *testing.B, repo contracts.TransactionalRepository[TestUser, contracts.ID], ctx context.Context, setupUsers []TestUser) {
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
}

func benchmarkFindAll(b *testing.B, repo contracts.TransactionalRepository[TestUser, contracts.ID], ctx context.Context) {
	b.Run("FindAll", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := repo.FindAll(ctx, 100, 0)
			if err != nil {
				b.Errorf("findall failed: %v", err)
			}
		}
	})
}

func benchmarkExists(b *testing.B, repo contracts.TransactionalRepository[TestUser, contracts.ID], ctx context.Context, setupUsers []TestUser) {
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
}

func benchmarkCount(b *testing.B, repo contracts.TransactionalRepository[TestUser, contracts.ID], ctx context.Context) {
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
	defer closeDatabaseWithLog(b, database)

	if err := createTestTable(database); err != nil {
		b.Fatalf("failed to create table: %v", err)
	}

	sqlDB := database.(*sqlDatabase).db
	mapper := &TestUserStrategyMapper{TestUserMapper: &TestUserMapper{}}
	ctx := context.Background()

	setupUsers := setupStrategyBenchmarkUsers(ctx)
	benchmarkStrategies(b, sqlDB, mapper, ctx, setupUsers)
}

func setupStrategyBenchmarkUsers(_ context.Context) []TestUser {
	setupUsers := make([]TestUser, 100)
	for i := 0; i < 100; i++ {
		user := NewTestUser(int64(i+1), fmt.Sprintf("User %d", i), fmt.Sprintf("user%d@example.com", i))
		setupUsers[i] = user
	}
	return setupUsers
}

func benchmarkStrategies(b *testing.B, sqlDB *sql.DB, mapper *TestUserStrategyMapper, ctx context.Context, setupUsers []TestUser) {
	strategies := []contracts.LoadingStrategy{
		contracts.LoadingStrategyMultiple,
		contracts.LoadingStrategyJoin,
		contracts.LoadingStrategyBatch,
	}

	for _, strategy := range strategies {
		repo := NewStrategyRepository[TestUser, contracts.ID, TestUserMemento](sqlDB, mapper, strategy)

		for _, user := range setupUsers {
			if err := repo.Save(ctx, user); err != nil {
				b.Skipf("failed to setup user with strategy %v: %v", strategy, err)
			}
		}

		benchmarkStrategyFind(b, repo, ctx, setupUsers, strategy)
		benchmarkStrategyFindAll(b, repo, ctx, strategy)
	}
}

func benchmarkStrategyFind(b *testing.B, repo contracts.StrategyRepository[TestUser, contracts.ID], ctx context.Context, setupUsers []TestUser, strategy contracts.LoadingStrategy) {
	b.Run(fmt.Sprintf("Find_%s", strategy), func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			idx := i % len(setupUsers)
			_, err := repo.Find(ctx, setupUsers[idx].id)
			if err != nil {
				b.Errorf("find failed for strategy %v: %v", strategy, err)
			}
		}
	})
}

func benchmarkStrategyFindAll(b *testing.B, repo contracts.StrategyRepository[TestUser, contracts.ID], ctx context.Context, strategy contracts.LoadingStrategy) {
	b.Run(fmt.Sprintf("FindAll_%s", strategy), func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := repo.FindAll(ctx, 50, 0)
			if err != nil {
				b.Errorf("findall failed for strategy %v: %v", strategy, err)
			}
		}
	})
}

func BenchmarkMigrationRunner(b *testing.B) {
	b.Run("Migration execution", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			database := setupTestDatabaseForBenchmark(b)

			defer func(db contracts.Database) {
				if err := db.Close(); err != nil {
					b.Logf("failed to close database during migration benchmark: %v", err)
				}
			}(database)

			migration := CreateMigration(fmt.Sprintf("bench_%d", i), "benchmark migration").
				CreateTable(fmt.Sprintf("bench_table_%d", i), "id INTEGER PRIMARY KEY", "data TEXT").
				Build()

			err := database.Migrate([]contracts.Migration{migration})
			if err != nil {
				b.Errorf("migration failed on iteration %d: %v", i, err)
			}
		}
	})
}

func BenchmarkQueryBuilder(b *testing.B) {
	benchmarkSimpleQueryBuilding(b)
	benchmarkComplexQueryBuilding(b)
}

func benchmarkSimpleQueryBuilding(b *testing.B) {
	b.Run("Simple query building", func(b *testing.B) {
		qb := NewQueryBuilder()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			qb.Reset()
			qb.Select("*").From("users").Where("id = ?", i).Build()
		}
	})
}

func benchmarkComplexQueryBuilding(b *testing.B) {
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
	defer closeDatabaseWithLog(b, database)

	sqlDB := database.(*sqlDatabase).db
	tm := NewTransactionManager(sqlDB)

	_, err := sqlDB.Exec("CREATE TABLE IF NOT EXISTS bench_tx (id INTEGER PRIMARY KEY, value TEXT)")
	if err != nil {
		b.Fatalf("failed to create bench_tx table: %v", err)
	}

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

func closeDatabaseWithLog(tb testing.TB, database contracts.Database) {
	if err := database.Close(); err != nil {
		tb.Logf("failed to close database: %v", err)
	}
}

func createTestTable(database contracts.Database) error {
	migration := CreateMigration("001", "create users").
		CreateTable("users", "id INTEGER PRIMARY KEY", "name TEXT", "email TEXT").
		Build()

	return database.Migrate([]contracts.Migration{migration})
}
