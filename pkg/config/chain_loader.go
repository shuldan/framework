package config

type chainLoader struct {
	loaders []Loader
}

func (c *chainLoader) Load() (map[string]any, error) {
	final := make(map[string]any)
	var lastErr error

	for _, loader := range c.loaders {
		config, err := loader.Load()
		if err != nil {
			lastErr = err
			continue
		}

		if err = mergeMaps(final, config); err != nil {
			return nil, ErrMergeFailed.
				WithCause(err)
		}
	}

	if len(final) == 0 {
		return nil, ErrNoConfigSource.WithCause(lastErr)
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
