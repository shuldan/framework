package config

type Loader interface {
	Load() (map[string]any, error)
}
