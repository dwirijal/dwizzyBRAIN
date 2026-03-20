package config

import (
	"fmt"
	"os"
	"strings"
)

// ReadOptional returns the env value for name or, if unset, the contents of
// name_FILE. It trims surrounding whitespace and returns an empty string when
// neither source is configured.
func ReadOptional(name string) (string, error) {
	return read(name, false)
}

// ReadRequired returns the env value for name or, if unset, the contents of
// name_FILE. It returns an error when neither source is configured.
func ReadRequired(name string) (string, error) {
	return read(name, true)
}

func read(name string, required bool) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("secret name is required")
	}

	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value, nil
	}

	fileName := strings.TrimSpace(os.Getenv(name + "_FILE"))
	if fileName != "" {
		raw, err := os.ReadFile(fileName)
		if err != nil {
			return "", fmt.Errorf("read %s from %s: %w", name, fileName, err)
		}
		return strings.TrimSpace(string(raw)), nil
	}

	if required {
		return "", fmt.Errorf("%s is required", name)
	}
	return "", nil
}
