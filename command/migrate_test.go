package command

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"testing"
	"time"

	"github.com/shuldan/cli"
	"github.com/shuldan/migrator"

	"github.com/shuldan/framework/database"
	"github.com/shuldan/framework/migration"
)

func init() {
	sql.Register("cmdtestdb", &cmdTestDBDriver{})
}

func TestMigrateUp_Metadata(t *testing.T) {
	t.Parallel()
	cmd := MigrateUp(nil)
	assertCommand(t, cmd, "migrate:up", databaseGroup)
	assertHasOption(t, cmd, "connection")
	if cmd.Args() != nil {
		t.Error("expected nil args")
	}
}

func TestMigrateDown_Metadata(t *testing.T) {
	t.Parallel()
	cmd := MigrateDown(nil)
	assertCommand(t, cmd, "migrate:down", databaseGroup)
	assertHasOption(t, cmd, "steps")
	assertHasOption(t, cmd, "force")
	assertHasOption(t, cmd, "connection")
	if cmd.Args() != nil {
		t.Error("expected nil args")
	}
}

func TestMigrateStatus_Metadata(t *testing.T) {
	t.Parallel()
	cmd := MigrateStatus(nil)
	assertCommand(t, cmd, "migrate:status", databaseGroup)
	assertHasOption(t, cmd, "connection")
	if cmd.Args() != nil {
		t.Error("expected nil args")
	}
}

func TestMigratePlan_Metadata(t *testing.T) {
	t.Parallel()
	cmd := MigratePlan(nil)
	assertCommand(t, cmd, "migrate:plan", databaseGroup)
	assertHasOption(t, cmd, "connection")
	if cmd.Args() != nil {
		t.Error("expected nil args")
	}
}

func TestMigrateCommands_AllConnectionOption(t *testing.T) {
	t.Parallel()
	runner := migration.NewRunner(nil, nil)
	cmds := []cli.Command{
		MigrateUp(runner), MigrateDown(runner),
		MigrateStatus(runner), MigratePlan(runner),
	}
	for _, cmd := range cmds {
		t.Run(cmd.Name(), func(t *testing.T) {
			assertHasOption(t, cmd, "connection")
		})
	}
}

func TestMigrateUp_Execute_Success(t *testing.T) {
	t.Parallel()
	runner := migration.NewRunner(nil, nil)
	output, err := runCommand(t, MigrateUp(runner))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, output, "Migrations applied successfully")
}

func TestMigrateDown_Execute_Success(t *testing.T) {
	t.Parallel()
	runner := migration.NewRunner(nil, nil)
	output, err := runCommand(t, MigrateDown(runner))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, output, "Migrations rolled back successfully")
}

func TestMigrateStatus_Execute_Success(t *testing.T) {
	t.Parallel()
	runner := migration.NewRunner(nil, nil)
	_, err := runCommand(t, MigrateStatus(runner))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMigratePlan_Execute_Success(t *testing.T) {
	t.Parallel()
	runner := migration.NewRunner(nil, nil)
	_, err := runCommand(t, MigratePlan(runner))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMigrateUp_Execute_Error(t *testing.T) {
	runner := newCmdTestRunner(t)
	_, err := runCommand(t, MigrateUp(runner), "--connection=default")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMigrateDown_Execute_Error(t *testing.T) {
	runner := newCmdTestRunner(t)
	_, err := runCommand(t, MigrateDown(runner), "--connection=default")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMigrateStatus_Execute_Error(t *testing.T) {
	runner := newCmdTestRunner(t)
	_, err := runCommand(t, MigrateStatus(runner), "--connection=default")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMigratePlan_Execute_Error(t *testing.T) {
	runner := newCmdTestRunner(t)
	_, err := runCommand(t, MigratePlan(runner), "--connection=default")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteStatusResults_MultipleConnections(t *testing.T) {
	t.Parallel()
	now := time.Now()
	batch := 1
	results := []migration.StatusResult{
		{
			Connection: "default",
			Migrations: []migrator.MigrationStatus{
				{ID: "001", State: migrator.MigrationStateApplied, AppliedAt: &now, Batch: batch},
			},
		},
		{Connection: "analytics", Migrations: nil},
	}
	var buf bytes.Buffer
	writeStatusResults(&buf, results)
	out := buf.String()
	assertContains(t, out, "default")
	assertContains(t, out, "analytics")
	assertContains(t, out, "No migrations registered")
}

func TestWriteStatusLine_Applied(t *testing.T) {
	t.Parallel()
	now := time.Now()
	batch := 2
	m := migrator.MigrationStatus{
		ID: "001", State: migrator.MigrationStateApplied, AppliedAt: &now, Batch: batch,
	}
	var buf bytes.Buffer
	writeStatusLine(&buf, m)
	out := buf.String()
	assertContains(t, out, "001")
	assertContains(t, out, fmt.Sprintf("%d", batch))
}

func TestWriteStatusLine_Pending(t *testing.T) {
	t.Parallel()
	m := migrator.MigrationStatus{
		ID: "002", State: migrator.MigrationStatePending, AppliedAt: nil,
	}
	var buf bytes.Buffer
	writeStatusLine(&buf, m)
	assertContains(t, buf.String(), "-")
}

func TestWritePlanResults_WithMigrations(t *testing.T) {
	t.Parallel()
	results := []migration.PlanResult{
		{
			Connection: "default",
			Migrations: []migrator.PlannedMigration{
				{ID: "001", Description: "create users", Queries: []string{"CREATE TABLE users;"}},
			},
		},
	}
	var buf bytes.Buffer
	writePlanResults(&buf, results)
	out := buf.String()
	assertContains(t, out, "default")
	assertContains(t, out, "001")
	assertContains(t, out, "CREATE TABLE users;")
}

func TestWritePlanResults_NoPending(t *testing.T) {
	t.Parallel()
	results := []migration.PlanResult{{Connection: "default", Migrations: nil}}
	var buf bytes.Buffer
	writePlanResults(&buf, results)
	assertContains(t, buf.String(), "No pending migrations")
}

func TestWritePlanResults_MultipleConnections(t *testing.T) {
	t.Parallel()
	results := []migration.PlanResult{
		{Connection: "first", Migrations: nil},
		{Connection: "second", Migrations: nil},
	}
	var buf bytes.Buffer
	writePlanResults(&buf, results)
	out := buf.String()
	assertContains(t, out, "first")
	assertContains(t, out, "second")
}

func TestWritePlanMigration(t *testing.T) {
	t.Parallel()
	m := migrator.PlannedMigration{
		ID: "003", Description: "add index", Queries: []string{"Q1", "Q2"},
	}
	var buf bytes.Buffer
	writePlanMigration(&buf, m)
	out := buf.String()
	assertContains(t, out, "003")
	assertContains(t, out, "add index")
	assertContains(t, out, "Q1")
	assertContains(t, out, "Q2")
}

func newCmdTestRunner(t *testing.T) *migration.Runner {
	t.Helper()
	configs := map[string]database.ConnectionConfig{
		"default": {Driver: "cmdtestdb", DSN: "test"},
	}
	mgr, err := database.NewManager(configs, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Stop(context.TODO()) })
	runner := migration.NewRunner(mgr, nil)
	runner.Register("default",
		migrator.CreateMigration("001", "test").RawUp("SELECT 1").MustBuild(),
	)
	return runner
}

func assertCommand(t *testing.T, cmd cli.Command, name, group string) {
	t.Helper()
	if cmd.Name() != name {
		t.Errorf("name: expected %q, got %q", name, cmd.Name())
	}
	if cmd.Group() != group {
		t.Errorf("group: expected %q, got %q", group, cmd.Group())
	}
	if cmd.Description() == "" {
		t.Error("description should not be empty")
	}
}

func assertHasOption(t *testing.T, cmd cli.Command, name string) {
	t.Helper()
	for _, opt := range cmd.Options() {
		if opt.Name == name {
			return
		}
	}
	t.Errorf("command %q: expected option %q", cmd.Name(), name)
}

type cmdTestDBDriver struct{}

func (d *cmdTestDBDriver) Open(_ string) (driver.Conn, error) {
	return &cmdTestDBConn{}, nil
}

type cmdTestDBConn struct{}

func (c *cmdTestDBConn) Prepare(_ string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (c *cmdTestDBConn) Close() error              { return nil }
func (c *cmdTestDBConn) Begin() (driver.Tx, error) { return nil, driver.ErrSkip }
