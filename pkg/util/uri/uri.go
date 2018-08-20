package uri

import (
	"net/url"

	"github.com/pkg/errors"
)

// ToPath converts URI to a filesystem path.
func ToPath(uriStr string) (string, error) {
	u, err := url.Parse(uriStr)
	if err != nil {
		return "", errors.Wrap(err, "parsing file URL")
	}

	if u.Scheme != "file" {
		return "", errors.Wrap(err, "invalid file schema")
	}

	return u.Path, nil
}
