package parser

import (
	"reflect"
	"testing"

	options "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	descriptor "google.golang.org/protobuf/types/descriptorpb"
)

func TestProto2Parser_ParseHTTPRules(t *testing.T) {
	t.Parallel()

	parser := NewProto2Parser()

	tests := []struct {
		name     string
		method   *descriptor.MethodDescriptorProto
		expected []HTTPRule
	}{
		{
			name: "method_without_options",
			method: &descriptor.MethodDescriptorProto{
				Name: proto.String("TestMethod"),
			},
			expected: []HTTPRule{},
		},
		{
			name: "method_with_get_rule",
			method: func() *descriptor.MethodDescriptorProto {
				method := &descriptor.MethodDescriptorProto{
					Name:    proto.String("GetUser"),
					Options: &descriptor.MethodOptions{},
				}

				httpRule := &options.HttpRule{
					Pattern: &options.HttpRule_Get{
						Get: "/v1/users/{id}",
					},
					Body: "",
				}

				proto.SetExtension(method.Options, options.E_Http, httpRule)
				return method
			}(),
			expected: []HTTPRule{
				{
					Method:     "GET",
					Pattern:    "/v1/users/{id}",
					Body:       "",
					PathParams: []string{"id"},
				},
			},
		},
		{
			name: "method_with_custom_rule",
			method: func() *descriptor.MethodDescriptorProto {
				method := &descriptor.MethodDescriptorProto{
					Name:    proto.String("CustomMethod"),
					Options: &descriptor.MethodOptions{},
				}

				httpRule := &options.HttpRule{
					Pattern: &options.HttpRule_Custom{
						Custom: &options.CustomHttpPattern{
							Kind: "CUSTOM",
							Path: "/custom/{id}/action",
						},
					},
					Body: "*",
				}

				proto.SetExtension(method.Options, options.E_Http, httpRule)
				return method
			}(),
			expected: []HTTPRule{
				{
					Method:     "CUSTOM",
					Pattern:    "/custom/{id}/action",
					Body:       "*",
					PathParams: []string{"id"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parser.ParseHTTPRules(tt.method)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseHTTPRules() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestProto2Parser_parseHTTPRule(t *testing.T) {
	t.Parallel()

	parser := NewProto2Parser()

	tests := []struct {
		name     string
		httpRule *options.HttpRule
		expected HTTPRule
	}{
		{
			name: "get_rule",
			httpRule: &options.HttpRule{
				Pattern: &options.HttpRule_Get{
					Get: "/v1/users/{id}",
				},
				Body: "",
			},
			expected: HTTPRule{
				Method:     "GET",
				Pattern:    "/v1/users/{id}",
				Body:       "",
				PathParams: []string{"id"},
			},
		},
		{
			name: "custom_rule",
			httpRule: &options.HttpRule{
				Pattern: &options.HttpRule_Custom{
					Custom: &options.CustomHttpPattern{
						Kind: "HEAD",
						Path: "/v1/users/{id}/status",
					},
				},
				Body: "",
			},
			expected: HTTPRule{
				Method:     "HEAD",
				Pattern:    "/v1/users/{id}/status",
				Body:       "",
				PathParams: []string{"id"},
			},
		},
		{
			name: "custom_rule_nil_custom",
			httpRule: &options.HttpRule{
				Pattern: &options.HttpRule_Custom{
					Custom: nil,
				},
				Body: "",
			},
			expected: HTTPRule{
				Method:     "",
				Pattern:    "",
				Body:       "",
				PathParams: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parser.parseHTTPRule(tt.httpRule)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseHTTPRule() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestProto2Parser_ParsePathParams(t *testing.T) {
	t.Parallel()

	parser := NewProto2Parser()

	tests := []struct {
		name     string
		pattern  string
		expected []string
	}{
		{
			name:     "empty_pattern",
			pattern:  "",
			expected: []string{},
		},
		{
			name:     "no_params",
			pattern:  "/v1/users",
			expected: []string{},
		},
		{
			name:     "single_param",
			pattern:  "/v1/users/{id}",
			expected: []string{"id"},
		},
		{
			name:     "multiple_params",
			pattern:  "/v1/users/{user_id}/posts/{post_id}",
			expected: []string{"user_id", "post_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parser.ParsePathParams(tt.pattern)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParsePathParams(%q) = %v, want %v", tt.pattern, result, tt.expected)
			}
		})
	}
}
