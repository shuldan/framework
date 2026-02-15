package migration

type RunnerOption func(*Runner)

func WithMigrationTable(name string) RunnerOption {
	return func(r *Runner) {
		r.tableName = name
	}
}

func WithAdvisoryLock() RunnerOption {
	return func(r *Runner) {
		r.lock = true
	}
}
