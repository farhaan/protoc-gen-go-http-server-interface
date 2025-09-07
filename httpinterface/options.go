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
}

// ParseOptions parses the parameter string from protoc into an Options struct
func ParseOptions(parameter string) (*Options, error) {
	options := &Options{}

	if parameter == "" {
		return options, nil
	}

	params := strings.Split(parameter, ",")
	for _, p := range params {
		if err := parseParameter(options, p); err != nil {
			return nil, err
		}
	}

	return options, nil
}

// parseParameter parses a single parameter key=value pair
func parseParameter(options *Options, param string) error {
	kv := strings.SplitN(param, "=", 2)
	if len(kv) != 2 {
		return fmt.Errorf("invalid parameter: %s", param)
	}

	key := strings.TrimSpace(kv[0])
	value := strings.TrimSpace(kv[1])

	switch key {
	case "paths":
		return parsePathsOption(options, value)
	case "output_prefix":
		options.OutputPrefix = value
		return nil
	default:
		return fmt.Errorf("unknown option: %s (valid options: paths, output_prefix)", key)
	}
}

// parsePathsOption parses the paths option
func parsePathsOption(options *Options, value string) error {
	switch value {
	case "source_relative":
		options.PathsSourceRelative = true
		return nil
	case "import":
		// Default behavior, no action needed
		return nil
	default:
		return fmt.Errorf("unknown paths option: %s", value)
	}
}
