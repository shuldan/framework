package config

import (
	"bytes"
	"os"
	"strings"
	"text/template"
)

type templatedLoader struct {
	loader Loader
}

func newTemplatedLoader(loader Loader) Loader {
	return &templatedLoader{
		loader: loader,
	}
}

func (t *templatedLoader) Load() (map[string]any, error) {
	raw, err := t.loader.Load()
	if err != nil {
		return nil, err
	}

	processed := make(map[string]any)
	for k, v := range raw {
		processed[k] = t.processValue(v)
	}
	return processed, nil
}

func (t *templatedLoader) processValue(v any) any {
	switch val := v.(type) {
	case string:
		if strings.Contains(val, "{{") && strings.Contains(val, "}}") {
			result, _ := t.render(val)
			return result
		}
		return val
	case map[string]any:
		mapped := make(map[string]any)
		for k, v := range val {
			mapped[k] = t.processValue(v)
		}
		return mapped
	case []any:
		var result []any
		for _, item := range val {
			result = append(result, t.processValue(item))
		}
		return result
	default:
		return val
	}
}

func (t *templatedLoader) newFuncMap() template.FuncMap {
	return template.FuncMap{
		"default": func(def, val interface{}) string {
			s, ok := val.(string)
			if !ok || s == "" {
				if s, ok := def.(string); ok {
					return s
				}
				return ""
			}
			return s
		},
		"env":   os.Getenv,
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
	}
}

func (t *templatedLoader) render(input string) (string, error) {
	tmpl, err := template.New("config").Funcs(t.newFuncMap()).Parse(input)
	if err != nil {
		return "", err
	}

	data := make(map[string]string)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			data[parts[0]] = parts[1]
		}
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
