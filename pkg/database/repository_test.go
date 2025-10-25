package database

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
)

type TestAggregate struct {
	id   TestID
	name string
}

func (a TestAggregate) ID() contracts.ID {
	return a.id
}

type TestID struct {
	value string
}

func (id TestID) String() string {
	return id.value
}

func (id TestID) IsValid() bool {
	return id.value != ""
}

type TestMemento struct {
	id   TestID
	name string
}

func (m TestMemento) GetID() contracts.ID {
	return m.id
}

type mockMapper struct {
	findFunc        func(context.Context, *sql.DB, contracts.ID) *sql.Row
	findAllFunc     func(context.Context, *sql.DB, int, int) (*sql.Rows, error)
	findByFunc      func(context.Context, *sql.DB, string, []any) (*sql.Rows, error)
	existsByFunc    func(context.Context, *sql.DB, string, []any) (bool, error)
	countByFunc     func(context.Context, *sql.DB, string, []any) (int64, error)
	saveFunc        func(context.Context, *sql.DB, TestMemento) error
	deleteFunc      func(context.Context, *sql.DB, contracts.ID) error
	toMementoFunc   func(TestAggregate) (TestMemento, error)
	fromMementoFunc func(TestMemento) (TestAggregate, error)
	fromRowFunc     func(*sql.Row) (TestMemento, error)
	fromRowsFunc    func(*sql.Rows) ([]TestMemento, error)
}

func (m *mockMapper) Find(ctx context.Context, db *sql.DB, id contracts.ID) *sql.Row {
	if m.findFunc != nil {
		return m.findFunc(ctx, db, id)
	}
	return nil
}

func (m *mockMapper) FindAll(ctx context.Context, db *sql.DB, limit, offset int) (*sql.Rows, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc(ctx, db, limit, offset)
	}
	return nil, nil
}

func (m *mockMapper) FindBy(ctx context.Context, db *sql.DB, conditions string, args []any) (*sql.Rows, error) {
	if m.findByFunc != nil {
		return m.findByFunc(ctx, db, conditions, args)
	}
	return nil, nil
}

func (m *mockMapper) ExistsBy(ctx context.Context, db *sql.DB, conditions string, args []any) (bool, error) {
	if m.existsByFunc != nil {
		return m.existsByFunc(ctx, db, conditions, args)
	}
	return false, nil
}

func (m *mockMapper) CountBy(ctx context.Context, db *sql.DB, conditions string, args []any) (int64, error) {
	if m.countByFunc != nil {
		return m.countByFunc(ctx, db, conditions, args)
	}
	return 0, nil
}

func (m *mockMapper) Save(ctx context.Context, db *sql.DB, memento TestMemento) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, db, memento)
	}
	return nil
}

func (m *mockMapper) Delete(ctx context.Context, db *sql.DB, id contracts.ID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, db, id)
	}
	return nil
}

func (m *mockMapper) ToMemento(aggregate TestAggregate) (TestMemento, error) {
	if m.toMementoFunc != nil {
		return m.toMementoFunc(aggregate)
	}
	return TestMemento{}, nil
}

func (m *mockMapper) FromMemento(memento TestMemento) (TestAggregate, error) {
	if m.fromMementoFunc != nil {
		return m.fromMementoFunc(memento)
	}
	return TestAggregate{}, nil
}

func (m *mockMapper) FromRow(row *sql.Row) (TestMemento, error) {
	if m.fromRowFunc != nil {
		return m.fromRowFunc(row)
	}
	return TestMemento{}, nil
}

func (m *mockMapper) FromRows(rows *sql.Rows) ([]TestMemento, error) {
	if m.fromRowsFunc != nil {
		return m.fromRowsFunc(rows)
	}
	return nil, nil
}

func TestRepository_Find(t *testing.T) {
	t.Parallel()

	t.Run("successful find", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		mapper := &mockMapper{
			findFunc: func(ctx context.Context, db *sql.DB, id contracts.ID) *sql.Row {
				return &sql.Row{}
			},
			fromRowFunc: func(row *sql.Row) (TestMemento, error) {
				return TestMemento{id: TestID{value: "1"}, name: "test"}, nil
			},
			fromMementoFunc: func(memento TestMemento) (TestAggregate, error) {
				return TestAggregate{id: memento.id, name: memento.name}, nil
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		aggregate, err := repo.Find(context.Background(), TestID{value: "1"})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if aggregate.ID().String() != "1" {
			t.Errorf("expected ID '1', got %s", aggregate.ID().String())
		}
		if aggregate.name != "test" {
			t.Errorf("expected name 'test', got %s", aggregate.name)
		}
	})

	t.Run("entity not found", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		mapper := &mockMapper{
			findFunc: func(ctx context.Context, db *sql.DB, id contracts.ID) *sql.Row {
				return &sql.Row{}
			},
			fromRowFunc: func(row *sql.Row) (TestMemento, error) {
				return TestMemento{}, sql.ErrNoRows
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		_, err := repo.Find(context.Background(), TestID{value: "1"})

		if !errors.Is(err, ErrEntityNotFound) {
			t.Errorf("expected ErrEntityNotFound, got %v", err)
		}
	})

	t.Run("mapper error", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		testErr := errors.New("mapper error")
		mapper := &mockMapper{
			findFunc: func(ctx context.Context, db *sql.DB, id contracts.ID) *sql.Row {
				return &sql.Row{}
			},
			fromRowFunc: func(row *sql.Row) (TestMemento, error) {
				return TestMemento{}, testErr
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		_, err := repo.Find(context.Background(), TestID{value: "1"})

		if !errors.Is(err, testErr) {
			t.Errorf("expected mapper error, got %v", err)
		}
	})

	t.Run("conversion error", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		testErr := errors.New("conversion error")
		mapper := &mockMapper{
			findFunc: func(ctx context.Context, db *sql.DB, id contracts.ID) *sql.Row {
				return &sql.Row{}
			},
			fromRowFunc: func(row *sql.Row) (TestMemento, error) {
				return TestMemento{id: TestID{value: "1"}, name: "test"}, nil
			},
			fromMementoFunc: func(memento TestMemento) (TestAggregate, error) {
				return TestAggregate{}, testErr
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		_, err := repo.Find(context.Background(), TestID{value: "1"})

		if !errors.Is(err, testErr) {
			t.Errorf("expected conversion error, got %v", err)
		}
	})
}

func TestRepository_FindAll(t *testing.T) {
	t.Parallel()

	t.Run("successful find all", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		mapper := &mockMapper{
			findAllFunc: func(ctx context.Context, db *sql.DB, limit, offset int) (*sql.Rows, error) {
				return &sql.Rows{}, nil
			},
			fromRowsFunc: func(rows *sql.Rows) ([]TestMemento, error) {
				return []TestMemento{
					{id: TestID{value: "1"}, name: "test1"},
					{id: TestID{value: "2"}, name: "test2"},
				}, nil
			},
			fromMementoFunc: func(memento TestMemento) (TestAggregate, error) {
				return TestAggregate{id: memento.id, name: memento.name}, nil
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		aggregates, err := repo.FindAll(context.Background(), 10, 0)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(aggregates) != 2 {
			t.Errorf("expected 2 aggregates, got %d", len(aggregates))
		}
	})

	t.Run("empty result", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		mapper := &mockMapper{
			findAllFunc: func(ctx context.Context, db *sql.DB, limit, offset int) (*sql.Rows, error) {
				return &sql.Rows{}, nil
			},
			fromRowsFunc: func(rows *sql.Rows) ([]TestMemento, error) {
				return []TestMemento{}, nil
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		aggregates, err := repo.FindAll(context.Background(), 10, 0)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(aggregates) != 0 {
			t.Errorf("expected 0 aggregates, got %d", len(aggregates))
		}
	})

	t.Run("mapper error", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		testErr := errors.New("mapper error")
		mapper := &mockMapper{
			findAllFunc: func(ctx context.Context, db *sql.DB, limit, offset int) (*sql.Rows, error) {
				return nil, testErr
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		_, err := repo.FindAll(context.Background(), 10, 0)

		if !errors.Is(err, testErr) {
			t.Errorf("expected mapper error, got %v", err)
		}
	})

	t.Run("conversion error", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		testErr := errors.New("conversion error")
		mapper := &mockMapper{
			findAllFunc: func(ctx context.Context, db *sql.DB, limit, offset int) (*sql.Rows, error) {
				return &sql.Rows{}, nil
			},
			fromRowsFunc: func(rows *sql.Rows) ([]TestMemento, error) {
				return []TestMemento{{id: TestID{value: "1"}, name: "test"}}, nil
			},
			fromMementoFunc: func(memento TestMemento) (TestAggregate, error) {
				return TestAggregate{}, testErr
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		_, err := repo.FindAll(context.Background(), 10, 0)

		if !errors.Is(err, testErr) {
			t.Errorf("expected conversion error, got %v", err)
		}
	})
}

func TestRepository_FindBy(t *testing.T) {
	t.Parallel()

	t.Run("successful find by", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		mapper := &mockMapper{
			findByFunc: func(ctx context.Context, db *sql.DB, conditions string, args []any) (*sql.Rows, error) {
				return &sql.Rows{}, nil
			},
			fromRowsFunc: func(rows *sql.Rows) ([]TestMemento, error) {
				return []TestMemento{{id: TestID{value: "1"}, name: "test"}}, nil
			},
			fromMementoFunc: func(memento TestMemento) (TestAggregate, error) {
				return TestAggregate{id: memento.id, name: memento.name}, nil
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		aggregates, err := repo.FindBy(context.Background(), "name = ?", []any{"test"})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(aggregates) != 1 {
			t.Errorf("expected 1 aggregate, got %d", len(aggregates))
		}
	})

	t.Run("no matches", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		mapper := &mockMapper{
			findByFunc: func(ctx context.Context, db *sql.DB, conditions string, args []any) (*sql.Rows, error) {
				return &sql.Rows{}, nil
			},
			fromRowsFunc: func(rows *sql.Rows) ([]TestMemento, error) {
				return []TestMemento{}, nil
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		aggregates, err := repo.FindBy(context.Background(), "name = ?", []any{"nonexistent"})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(aggregates) != 0 {
			t.Errorf("expected 0 aggregates, got %d", len(aggregates))
		}
	})

	t.Run("mapper error", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		testErr := errors.New("mapper error")
		mapper := &mockMapper{
			findByFunc: func(ctx context.Context, db *sql.DB, conditions string, args []any) (*sql.Rows, error) {
				return nil, testErr
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		_, err := repo.FindBy(context.Background(), "name = ?", []any{"test"})

		if !errors.Is(err, testErr) {
			t.Errorf("expected mapper error, got %v", err)
		}
	})

	t.Run("conversion error", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		testErr := errors.New("conversion error")
		mapper := &mockMapper{
			findByFunc: func(ctx context.Context, db *sql.DB, conditions string, args []any) (*sql.Rows, error) {
				return &sql.Rows{}, nil
			},
			fromRowsFunc: func(rows *sql.Rows) ([]TestMemento, error) {
				return []TestMemento{{id: TestID{value: "1"}, name: "test"}}, nil
			},
			fromMementoFunc: func(memento TestMemento) (TestAggregate, error) {
				return TestAggregate{}, testErr
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		_, err := repo.FindBy(context.Background(), "name = ?", []any{"test"})

		if !errors.Is(err, testErr) {
			t.Errorf("expected conversion error, got %v", err)
		}
	})
}

func TestRepository_ExistsBy(t *testing.T) {
	t.Parallel()

	t.Run("condition exists", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		mapper := &mockMapper{
			existsByFunc: func(ctx context.Context, db *sql.DB, conditions string, args []any) (bool, error) {
				return true, nil
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		exists, err := repo.ExistsBy(context.Background(), "name = ?", []any{"test"})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if !exists {
			t.Error("expected true, got false")
		}
	})

	t.Run("condition does not exist", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		mapper := &mockMapper{
			existsByFunc: func(ctx context.Context, db *sql.DB, conditions string, args []any) (bool, error) {
				return false, nil
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		exists, err := repo.ExistsBy(context.Background(), "name = ?", []any{"nonexistent"})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if exists {
			t.Error("expected false, got true")
		}
	})

	t.Run("mapper error", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		testErr := errors.New("mapper error")
		mapper := &mockMapper{
			existsByFunc: func(ctx context.Context, db *sql.DB, conditions string, args []any) (bool, error) {
				return false, testErr
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		_, err := repo.ExistsBy(context.Background(), "name = ?", []any{"test"})

		if !errors.Is(err, testErr) {
			t.Errorf("expected mapper error, got %v", err)
		}
	})
}

func TestRepository_CountBy(t *testing.T) {
	t.Parallel()

	t.Run("successful count", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		mapper := &mockMapper{
			countByFunc: func(ctx context.Context, db *sql.DB, conditions string, args []any) (int64, error) {
				return 5, nil
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		count, err := repo.CountBy(context.Background(), "active = ?", []any{true})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if count != 5 {
			t.Errorf("expected count 5, got %d", count)
		}
	})

	t.Run("zero count", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		mapper := &mockMapper{
			countByFunc: func(ctx context.Context, db *sql.DB, conditions string, args []any) (int64, error) {
				return 0, nil
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		count, err := repo.CountBy(context.Background(), "active = ?", []any{false})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if count != 0 {
			t.Errorf("expected count 0, got %d", count)
		}
	})

	t.Run("mapper error", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		testErr := errors.New("mapper error")
		mapper := &mockMapper{
			countByFunc: func(ctx context.Context, db *sql.DB, conditions string, args []any) (int64, error) {
				return 0, testErr
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		_, err := repo.CountBy(context.Background(), "active = ?", []any{true})

		if !errors.Is(err, testErr) {
			t.Errorf("expected mapper error, got %v", err)
		}
	})
}

func TestRepository_Save(t *testing.T) {
	t.Parallel()

	t.Run("successful save", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		mapper := &mockMapper{
			toMementoFunc: func(aggregate TestAggregate) (TestMemento, error) {
				return TestMemento{id: aggregate.id, name: aggregate.name}, nil
			},
			saveFunc: func(ctx context.Context, db *sql.DB, memento TestMemento) error {
				return nil
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		aggregate := TestAggregate{id: TestID{value: "1"}, name: "test"}
		err := repo.Save(context.Background(), aggregate)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("conversion error", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		testErr := errors.New("conversion error")
		mapper := &mockMapper{
			toMementoFunc: func(aggregate TestAggregate) (TestMemento, error) {
				return TestMemento{}, testErr
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		aggregate := TestAggregate{id: TestID{value: "1"}, name: "test"}
		err := repo.Save(context.Background(), aggregate)

		if !errors.Is(err, testErr) {
			t.Errorf("expected conversion error, got %v", err)
		}
	})

	t.Run("mapper error", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		testErr := errors.New("mapper error")
		mapper := &mockMapper{
			toMementoFunc: func(aggregate TestAggregate) (TestMemento, error) {
				return TestMemento{id: aggregate.id, name: aggregate.name}, nil
			},
			saveFunc: func(ctx context.Context, db *sql.DB, memento TestMemento) error {
				return testErr
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		aggregate := TestAggregate{id: TestID{value: "1"}, name: "test"}
		err := repo.Save(context.Background(), aggregate)

		if !errors.Is(err, testErr) {
			t.Errorf("expected mapper error, got %v", err)
		}
	})
}

func TestRepository_Delete(t *testing.T) {
	t.Parallel()

	t.Run("successful delete", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		mapper := &mockMapper{
			deleteFunc: func(ctx context.Context, db *sql.DB, id contracts.ID) error {
				return nil
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		err := repo.Delete(context.Background(), TestID{value: "1"})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("mapper error", func(t *testing.T) {
		t.Parallel()
		db := &sql.DB{}
		testErr := errors.New("mapper error")
		mapper := &mockMapper{
			deleteFunc: func(ctx context.Context, db *sql.DB, id contracts.ID) error {
				return testErr
			},
		}

		repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)
		err := repo.Delete(context.Background(), TestID{value: "1"})

		if !errors.Is(err, testErr) {
			t.Errorf("expected mapper error, got %v", err)
		}
	})
}

func TestRepository_Concurrency(t *testing.T) {
	db := &sql.DB{}
	callCount := 0
	mu := sync.Mutex{}
	mapper := &mockMapper{
		findFunc: func(ctx context.Context, db *sql.DB, id contracts.ID) *sql.Row {
			mu.Lock()
			callCount++
			mu.Unlock()
			return &sql.Row{}
		},
		fromRowFunc: func(row *sql.Row) (TestMemento, error) {
			return TestMemento{id: TestID{value: "1"}, name: "test"}, nil
		},
		fromMementoFunc: func(memento TestMemento) (TestAggregate, error) {
			return TestAggregate{id: memento.id, name: memento.name}, nil
		},
	}

	repo := NewRepository[TestAggregate, TestID, TestMemento](db, mapper)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = repo.Find(context.Background(), TestID{value: "1"})
		}()
	}
	wg.Wait()

	if callCount != 10 {
		t.Errorf("expected 10 calls, got %d", callCount)
	}
}

func TestRepository_Integration(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	_, err = db.Exec(`CREATE TABLE test_entities (id TEXT PRIMARY KEY, name TEXT)`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	mapper := &testEntityMapper{}
	repo := NewRepository[TestAggregate, TestID, testEntityMemento](db, mapper)

	aggregate := TestAggregate{id: TestID{value: "1"}, name: "test entity"}
	err = repo.Save(context.Background(), aggregate)
	if err != nil {
		t.Fatalf("failed to save entity: %v", err)
	}

	found, err := repo.Find(context.Background(), TestID{value: "1"})
	if err != nil {
		t.Fatalf("failed to find entity: %v", err)
	}
	if found.name != "test entity" {
		t.Errorf("expected name 'test entity', got %s", found.name)
	}

	all, err := repo.FindAll(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("failed to find all entities: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 entity, got %d", len(all))
	}

	filtered, err := repo.FindBy(context.Background(), "name = ?", []any{"test entity"})
	if err != nil {
		t.Fatalf("failed to find by condition: %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("expected 1 entity, got %d", len(filtered))
	}

	exists, err := repo.ExistsBy(context.Background(), "name = ?", []any{"test entity"})
	if err != nil {
		t.Fatalf("failed to check existence: %v", err)
	}
	if !exists {
		t.Error("expected entity to exist")
	}

	count, err := repo.CountBy(context.Background(), "name = ?", []any{"test entity"})
	if err != nil {
		t.Fatalf("failed to count entities: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	err = repo.Delete(context.Background(), TestID{value: "1"})
	if err != nil {
		t.Fatalf("failed to delete entity: %v", err)
	}

	_, err = repo.Find(context.Background(), TestID{value: "1"})
	if !errors.Is(err, ErrEntityNotFound) {
		t.Errorf("expected ErrEntityNotFound after deletion, got %v", err)
	}
}

type testEntityMemento struct {
	id   string
	name string
}

func (t testEntityMemento) GetID() contracts.ID {
	return NewStringID(t.id)
}

type testEntityMapper struct{}

func (m *testEntityMapper) Find(ctx context.Context, db *sql.DB, id contracts.ID) *sql.Row {
	return db.QueryRowContext(ctx, "SELECT id, name FROM test_entities WHERE id = ?", id.String())
}

func (m *testEntityMapper) FindAll(ctx context.Context, db *sql.DB, limit, offset int) (*sql.Rows, error) {
	return db.QueryContext(ctx, "SELECT id, name FROM test_entities LIMIT ? OFFSET ?", limit, offset)
}

func (m *testEntityMapper) FindBy(ctx context.Context, db *sql.DB, conditions string, args []any) (*sql.Rows, error) {
	query := "SELECT id, name FROM test_entities WHERE " + conditions
	return db.QueryContext(ctx, query, args...)
}

func (m *testEntityMapper) ExistsBy(ctx context.Context, db *sql.DB, conditions string, args []any) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM test_entities WHERE " + conditions + ")"
	var exists bool
	err := db.QueryRowContext(ctx, query, args...).Scan(&exists)
	return exists, err
}

func (m *testEntityMapper) CountBy(ctx context.Context, db *sql.DB, conditions string, args []any) (int64, error) {
	query := "SELECT COUNT(*) FROM test_entities WHERE " + conditions
	var count int64
	err := db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

func (m *testEntityMapper) Save(ctx context.Context, db *sql.DB, memento testEntityMemento) error {
	_, err := db.ExecContext(ctx, "INSERT OR REPLACE INTO test_entities (id, name) VALUES (?, ?)", memento.id, memento.name)
	return err
}

func (m *testEntityMapper) Delete(ctx context.Context, db *sql.DB, id contracts.ID) error {
	_, err := db.ExecContext(ctx, "DELETE FROM test_entities WHERE id = ?", id.String())
	return err
}

func (m *testEntityMapper) ToMemento(aggregate TestAggregate) (testEntityMemento, error) {
	return testEntityMemento{id: aggregate.id.String(), name: aggregate.name}, nil
}

func (m *testEntityMapper) FromMemento(memento testEntityMemento) (TestAggregate, error) {
	return TestAggregate{id: TestID{value: memento.id}, name: memento.name}, nil
}

func (m *testEntityMapper) FromRow(row *sql.Row) (testEntityMemento, error) {
	var memento testEntityMemento
	err := row.Scan(&memento.id, &memento.name)
	return memento, err
}

func (m *testEntityMapper) FromRows(rows *sql.Rows) ([]testEntityMemento, error) {
	defer func() {
		_ = rows.Close()
	}()
	var mementos []testEntityMemento
	for rows.Next() {
		var memento testEntityMemento
		if err := rows.Scan(&memento.id, &memento.name); err != nil {
			return nil, err
		}
		mementos = append(mementos, memento)
	}
	return mementos, rows.Err()
}
