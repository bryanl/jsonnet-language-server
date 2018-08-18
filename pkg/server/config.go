package server

import "github.com/pkg/errors"

// Config is configuration setting for the server.
type Config struct {
	// JsonnetLibPaths are jsonnet lib paths.
	JsonnetLibPaths []string
}

func loadConfig(data interface{}) (*Config, error) {
	options, ok := data.(map[string]interface{})
	if !ok {
		return new(Config), nil
	}

	config := &Config{}

	if jpaths, ok := options["jpaths"].([]interface{}); ok {
		strs, err := interfaceToStrings(jpaths)
		if err != nil {
			return nil, err
		}
		config.JsonnetLibPaths = strs
	}

	return config, nil
}

func interfaceToStrings(items []interface{}) ([]string, error) {
	var out []string
	for _, item := range items {
		str, ok := item.(string)
		if !ok {
			return nil, errors.Errorf("item was not a string")
		}

		out = append(out, str)
	}

	return out, nil
}
