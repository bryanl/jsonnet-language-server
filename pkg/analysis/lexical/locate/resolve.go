package locate

import "github.com/pkg/errors"

func resolve(item interface{}) (string, error) {
	switch item.(type) {
	default:
		return "", errors.Errorf("unable to resolve %T", item)
	}
}
