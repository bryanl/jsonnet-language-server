package server

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

const (
	cfgJsonnetLibPaths = "jsonnet.libPaths"
)

// Config is configuration setting for the server.
type Config struct {
	// JsonnetLibPaths are jsonnet lib paths.
	JsonnetLibPaths []string
}

// NewConfig creates an instance of Config.
func NewConfig() *Config {
	return &Config{
		JsonnetLibPaths: make([]string, 0),
	}
}

// Update updates the configuration.
func (c *Config) Update(update map[string]interface{}) error {
	for k, v := range update {
		switch k {
		case cfgJsonnetLibPaths:
			paths, err := interfaceToStrings(v)
			if err != nil {
				return errors.Wrapf(err, "setting %q", cfgJsonnetLibPaths)
			}

			c.JsonnetLibPaths = paths
		default:
			return errors.Errorf("setting %q is unknown to the jsonnet language server", k)
		}
	}
	return nil
}

func (c *Config) String() string {
	data, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Sprintf("marshaling config to JSON: %v", err))
	}
	return string(data)
}

func interfaceToStrings(v interface{}) ([]string, error) {
	switch v := v.(type) {
	case []interface{}:
		var out []string
		for _, item := range v {
			str, ok := item.(string)
			if !ok {
				return nil, errors.Errorf("item was not a string")
			}

			out = append(out, str)
		}

		return out, nil
	case []string:
		return v, nil
	default:
		return nil, errors.Errorf("unable to convert %T to array of strings", v)
	}
}
