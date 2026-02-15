package command

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/shuldan/cli"
	"github.com/shuldan/config"
)

var sensitiveKeys = []string{
	"password", "secret", "token", "key", "dsn", "credential",
}

func ConfigDump(cfg *config.Config) cli.Command {
	return &configDumpCommand{cfg: cfg}
}

type configDumpCommand struct {
	cfg *config.Config
}

func (c *configDumpCommand) Name() string        { return "config:dump" }
func (c *configDumpCommand) Description() string { return "Display loaded configuration" }
func (c *configDumpCommand) Group() string       { return "debug" }
func (c *configDumpCommand) Args() []cli.Arg     { return nil }

func (c *configDumpCommand) Options() []cli.Option {
	return []cli.Option{
		cli.BoolOption("no-mask", "", false,
			"Show sensitive values unmasked"),
	}
}

func (c *configDumpCommand) Execute(
	_ context.Context,
	_ io.Reader, out io.Writer, input *cli.Input,
) error {
	noMask := input.BoolOption("no-mask")
	all := c.cfg.All()

	printMap(out, all, "", noMask)

	return nil
}

func printMap(
	w io.Writer, m map[string]any, prefix string,
	noMask bool,
) {
	keys := sortedMapKeys(m)

	for _, k := range keys {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}

		v := m[k]

		if sub, ok := v.(map[string]any); ok {
			_, _ = fmt.Fprintf(w, "%s:\n", k)
			printMap(w, sub, fullKey, noMask)
			continue
		}

		display := formatValue(fullKey, v, noMask)
		_, _ = fmt.Fprintf(w, "  %s: %s\n", k, display)
	}
}

func formatValue(
	key string, val any, noMask bool,
) string {
	s := fmt.Sprintf("%v", val)

	if !noMask && isSensitiveKey(key) {
		return "***"
	}

	return s
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, s := range sensitiveKeys {
		if strings.Contains(lower, s) {
			return true
		}
	}

	return false
}

func sortedMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}
