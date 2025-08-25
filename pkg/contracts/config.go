package contracts

type Config interface {
	Has(key string) bool

	Get(key string) any

	GetString(key string, defaultVal ...string) string

	GetInt(key string, defaultVal ...int) int

	GetInt64(key string, defaultVal ...int64) int64

	GetFloat64(key string, defaultVal ...float64) float64

	GetBool(key string, defaultVal ...bool) bool

	GetStringSlice(key string, separator ...string) []string

	GetSub(key string) (Config, bool)

	All() map[string]any
}
