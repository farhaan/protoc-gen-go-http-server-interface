package parser

import (
	"testing"

	"google.golang.org/protobuf/proto"
	descriptor "google.golang.org/protobuf/types/descriptorpb"
)

func TestCreateParser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		file     *descriptor.FileDescriptorProto
		expected string // parser type name
	}{
		{
			name: "proto2_file",
			file: &descriptor.FileDescriptorProto{
				Syntax: proto.String("proto2"),
			},
			expected: "*parser.Proto2Parser",
		},
		{
			name: "proto3_file",
			file: &descriptor.FileDescriptorProto{
				Syntax: proto.String("proto3"),
			},
			expected: "*parser.Proto3Parser",
		},
		{
			name:     "no_syntax_defaults_to_proto2",
			file:     &descriptor.FileDescriptorProto{},
			expected: "*parser.Proto2Parser",
		},
		{
			name: "editions_file_with_option",
			file: &descriptor.FileDescriptorProto{
				Options: &descriptor.FileOptions{
					UninterpretedOption: []*descriptor.UninterpretedOption{
						{
							Name: []*descriptor.UninterpretedOption_NamePart{
								{
									NamePart:    proto.String("edition"),
									IsExtension: proto.Bool(false),
								},
							},
						},
					},
				},
			},
			expected: "*parser.EditionsParser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parser := CreateParser(tt.file)

			// Use type assertion to get the actual type
			actualType := ""
			switch parser.(type) {
			case *Proto2Parser:
				actualType = "*parser.Proto2Parser"
			case *Proto3Parser:
				actualType = "*parser.Proto3Parser"
			case *EditionsParser:
				actualType = "*parser.EditionsParser"
			default:
				actualType = "unknown"
			}

			if actualType != tt.expected {
				t.Errorf("CreateParser() = %v, want %v", actualType, tt.expected)
			}
		})
	}
}

func TestHasEditionOption(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		file     *descriptor.FileDescriptorProto
		expected bool
	}{
		{
			name:     "no_options",
			file:     &descriptor.FileDescriptorProto{},
			expected: false,
		},
		{
			name: "options_but_no_edition",
			file: &descriptor.FileDescriptorProto{
				Options: &descriptor.FileOptions{
					UninterpretedOption: []*descriptor.UninterpretedOption{
						{
							Name: []*descriptor.UninterpretedOption_NamePart{
								{
									NamePart:    proto.String("other"),
									IsExtension: proto.Bool(false),
								},
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "has_edition_option",
			file: &descriptor.FileDescriptorProto{
				Options: &descriptor.FileOptions{
					UninterpretedOption: []*descriptor.UninterpretedOption{
						{
							Name: []*descriptor.UninterpretedOption_NamePart{
								{
									NamePart:    proto.String("edition"),
									IsExtension: proto.Bool(false),
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "multiple_options_with_edition",
			file: &descriptor.FileDescriptorProto{
				Options: &descriptor.FileOptions{
					UninterpretedOption: []*descriptor.UninterpretedOption{
						{
							Name: []*descriptor.UninterpretedOption_NamePart{
								{
									NamePart:    proto.String("other"),
									IsExtension: proto.Bool(false),
								},
							},
						},
						{
							Name: []*descriptor.UninterpretedOption_NamePart{
								{
									NamePart:    proto.String("edition"),
									IsExtension: proto.Bool(false),
								},
							},
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := hasEditionOption(tt.file)
			if result != tt.expected {
				t.Errorf("hasEditionOption() = %v, want %v", result, tt.expected)
			}
		})
	}
}
