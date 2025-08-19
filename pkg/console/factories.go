package console

func NewRegistry() Registry {
	return &cmdRegistry{
		commands: make(map[string]Command),
		groups:   make(map[string][]string),
	}
}

func New(registry Registry) (Console, error) {
	if registry == nil {
		registry = NewRegistry()
	}

	p := newParser(registry)
	e := newExecutor(p)

	c := &console{
		registry:    registry,
		cmdParser:   p,
		cmdExecutor: e,
	}

	return c, nil
}
