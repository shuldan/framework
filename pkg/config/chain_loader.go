package config

import (
	"github.com/shuldan/framework/pkg/errors"
)

type chainLoader struct {
	loaders []Loader
}

func NewChainLoader(loaders ...Loader) Loader {
	return &chainLoader{loaders: loaders}
}

func (c *chainLoader) Load() (map[string]any, error) {
	final := make(map[string]any)
	var lastErr error

	for _, loader := range c.loaders {
		config, err := loader.Load()
		if err != nil {
			if errors.Is(err, ErrNoConfigSource) {
				continue
			}
			lastErr = err
			continue
		}

		if err = mergeMaps(final, config); err != nil {
			return nil, ErrMergeFailed.WithCause(err)
		}
	}

	if len(final) == 0 && lastErr != nil {
		return nil, ErrNoConfigSource.WithDetail("loader", "chain").WithCause(lastErr)
	}

	return final, nil
}

func mergeMaps(dst, src map[string]any) error {
	for k, v := range src {
		if vMap, ok := v.(map[string]any); ok {
			if dstV, exists := dst[k]; exists {
				if dstMap, ok := dstV.(map[string]any); ok {
					err := mergeMaps(dstMap, vMap)
					if err != nil {
						return err
					}
					continue
				}
			}
		}
		dst[k] = v
	}
	return nil
}
