package httpinterface

import (
	"reflect"
	"testing"

	options "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	descriptor "google.golang.org/protobuf/types/descriptorpb"
)

func TestCreateHTTPRuleExtractorForFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		file     *descriptor.FileDescriptorProto
		method   *descriptor.MethodDescriptorProto
		expected []HTTPRule
	}{
		{
			name: "proto3_file_with_http_rule",
			file: &descriptor.FileDescriptorProto{
				Syntax: proto.String("proto3"),
			},
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
			name: "editions_file_with_http_rule",
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
			method: func() *descriptor.MethodDescriptorProto {
				method := &descriptor.MethodDescriptorProto{
					Name:    proto.String("UpdateUser"),
					Options: &descriptor.MethodOptions{},
				}

				httpRule := &options.HttpRule{
					Pattern: &options.HttpRule_Put{
						Put: "/v1/users/{user_id}",
					},
					Body: "user",
				}

				proto.SetExtension(method.Options, options.E_Http, httpRule)
				return method
			}(),
			expected: []HTTPRule{
				{
					Method:     "PUT",
					Pattern:    "/v1/users/{user_id}",
					Body:       "user",
					PathParams: []string{"user_id"},
				},
			},
		},
		{
			name: "method_without_http_rule",
			file: &descriptor.FileDescriptorProto{
				Syntax: proto.String("proto3"),
			},
			method: &descriptor.MethodDescriptorProto{
				Name: proto.String("TestMethod"),
			},
			expected: []HTTPRule{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			extractor := CreateHTTPRuleExtractorForFile(tt.file)
			result := extractor(tt.method)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("CreateHTTPRuleExtractorForFile() extractor result = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestCreatePathParamExtractorForFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		file     *descriptor.FileDescriptorProto
		pattern  string
		expected []string
	}{
		{
			name: "proto3_file",
			file: &descriptor.FileDescriptorProto{
				Syntax: proto.String("proto3"),
			},
			pattern:  "/v1/users/{id}/posts/{post_id}",
			expected: []string{"id", "post_id"},
		},
		{
			name: "editions_file",
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
			pattern:  "/v1/organizations/{org_id}/users/{user_id}",
			expected: []string{"org_id", "user_id"},
		},
		{
			name:     "proto2_file",
			file:     &descriptor.FileDescriptorProto{},
			pattern:  "/v1/simple/{param}",
			expected: []string{"param"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			extractor := CreatePathParamExtractorForFile(tt.file)
			result := extractor(tt.pattern)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("CreatePathParamExtractorForFile() extractor result = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCreatePathPatternConverterForFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		file     *descriptor.FileDescriptorProto
		pattern  string
		expected string
	}{
		{
			name: "proto3_file",
			file: &descriptor.FileDescriptorProto{
				Syntax: proto.String("proto3"),
			},
			pattern:  "/v1/users/{id}",
			expected: "/v1/users/{id}",
		},
		{
			name: "editions_file",
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
			pattern:  "/v1/organizations/{org_id}/users/{user_id}",
			expected: "/v1/organizations/{org_id}/users/{user_id}",
		},
		{
			name:     "proto2_file",
			file:     &descriptor.FileDescriptorProto{},
			pattern:  "/api/test",
			expected: "/api/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			converter := CreatePathPatternConverterForFile(tt.file)
			result := converter(tt.pattern)

			if result != tt.expected {
				t.Errorf("CreatePathPatternConverterForFile() converter result = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParseMethodHTTPRules_Legacy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		method   *descriptor.MethodDescriptorProto
		expected []HTTPRule
	}{
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
			name: "method_with_post_and_additional_bindings",
			method: func() *descriptor.MethodDescriptorProto {
				method := &descriptor.MethodDescriptorProto{
					Name:    proto.String("CreateUser"),
					Options: &descriptor.MethodOptions{},
				}

				httpRule := &options.HttpRule{
					Pattern: &options.HttpRule_Post{
						Post: "/v1/users",
					},
					Body: "user",
					AdditionalBindings: []*options.HttpRule{
						{
							Pattern: &options.HttpRule_Put{
								Put: "/v1/users/{id}",
							},
							Body: "user",
						},
					},
				}

				proto.SetExtension(method.Options, options.E_Http, httpRule)
				return method
			}(),
			expected: []HTTPRule{
				{
					Method:     "POST",
					Pattern:    "/v1/users",
					Body:       "user",
					PathParams: []string{},
				},
				{
					Method:     "PUT",
					Pattern:    "/v1/users/{id}",
					Body:       "user",
					PathParams: []string{"id"},
				},
			},
		},
		{
			name: "method_without_http_options",
			method: &descriptor.MethodDescriptorProto{
				Name: proto.String("NoHTTP"),
			},
			expected: []HTTPRule{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parseMethodHTTPRules(tt.method)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseMethodHTTPRules() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestParseHTTPRule_Legacy(t *testing.T) {
	t.Parallel()

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
				Method:  "GET",
				Pattern: "/v1/users/{id}",
				Body:    "",
			},
		},
		{
			name: "post_rule_with_body",
			httpRule: &options.HttpRule{
				Pattern: &options.HttpRule_Post{
					Post: "/v1/users",
				},
				Body: "user",
			},
			expected: HTTPRule{
				Method:  "POST",
				Pattern: "/v1/users",
				Body:    "user",
			},
		},
		{
			name: "custom_rule",
			httpRule: &options.HttpRule{
				Pattern: &options.HttpRule_Custom{
					Custom: &options.CustomHttpPattern{
						Kind: "SEARCH",
						Path: "/v1/search/{query}",
					},
				},
				Body: "*",
			},
			expected: HTTPRule{
				Method:  "SEARCH",
				Pattern: "/v1/search/{query}",
				Body:    "*",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parseHTTPRule(tt.httpRule)

			if result.Method != tt.expected.Method || result.Pattern != tt.expected.Pattern || result.Body != tt.expected.Body {
				t.Errorf("parseHTTPRule() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestParsePathParams_Legacy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pattern  string
		expected []string
	}{
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
		{
			name:     "param_with_dots",
			pattern:  "/v1/users/{user.id}",
			expected: []string{"user.id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parsePathParams(tt.pattern)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parsePathParams(%q) = %v, want %v", tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestConvertPathPattern_Legacy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pattern  string
		expected string
	}{
		{
			name:     "no_conversion",
			pattern:  "/v1/users/{id}",
			expected: "/v1/users/{id}",
		},
		{
			name:     "simple_path",
			pattern:  "/api/test",
			expected: "/api/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := convertPathPattern(tt.pattern)

			if result != tt.expected {
				t.Errorf("convertPathPattern(%q) = %q, want %q", tt.pattern, result, tt.expected)
			}
		})
	}
}
