package httpinterface

import (
	"fmt"
	"strings"
)

// Options represents the plugin options
type Options struct {
	// PathsSourceRelative determines if the output files should use source-relative paths
	PathsSourceRelative bool
	// OutputPrefix is an optional prefix for output files
	OutputPrefix string
	// SupportsEditions determines if the plugin should advertise editions support
	SupportsEditions bool
}

// ParseOptions parses the parameter string from protoc into an Options struct
func ParseOptions(parameter string) (*Options, error) {
	options := &Options{}

	if parameter == "" {
		return options, nil
	}

	params := strings.Split(parameter, ",")
	for _, p := range params {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid parameter: %s", p)
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		switch key {
		case "paths":
			if value == "source_relative" {
				options.PathsSourceRelative = true
			} else if value != "import" {
				return nil, fmt.Errorf("unknown paths option: %s", value)
			}
		case "output_prefix":
			options.OutputPrefix = value
		case "editions":
			if value == "false" {
				options.SupportsEditions = false
			} else if value != "true" {
				return nil, fmt.Errorf("invalid editions option: %s (must be 'true' or 'false')", value)
			}
		default:
			// You might want to either error on unknown options, or just ignore them
			// For now, we'll log a warning and continue
			fmt.Printf("Warning: unknown option: %s=%s\n", key, value)
		}
	}

	return options, nil
}
