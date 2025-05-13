package parser

import (
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

// HTTPRule represents an HTTP binding from annotations
type HTTPRule struct {
	Method     string
	Pattern    string
	Body       string
	PathParams []string
}

type Parser interface {
	// ParseHTTPRules extracts HTTP rules from a method descriptor
	ParseHTTPRules(method *descriptor.MethodDescriptorProto) []HTTPRule

	// ParsePathParams extracts path parameters from a URL pattern
	ParsePathParams(pattern string) []string

	// ConvertPathPattern converts a path pattern to Go format
	ConvertPathPattern(pattern string) string
}

// CreateParser creates a parser appropriate for the given FileDescriptorProto
func CreateParser(file *descriptor.FileDescriptorProto) Parser {
	// Check for edition option
	if hasEditionOption(file) {
		return NewEditionsParser()
	}

	// Check syntax field
	syntax := file.GetSyntax()
	if syntax == "proto3" {
		return NewProto3Parser()
	}

	// Default to proto2
	return NewProto2Parser()
}

// hasEditionOption checks if a file has the edition option set
func hasEditionOption(file *descriptor.FileDescriptorProto) bool {
	if file.Options == nil {
		return false
	}

	// This would need to be implemented to check for the edition option
	// in the uninterpreted options of the file
	// Check for edition in uninterpreted options
	for _, option := range file.Options.UninterpretedOption {
		for _, name := range option.Name {
			if name.GetNamePart() == "edition" {
				return true
			}
		}
	}

	return false
}
