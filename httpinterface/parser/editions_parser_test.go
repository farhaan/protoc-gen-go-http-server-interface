package parser

import (
	"reflect"
	"testing"

	options "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	descriptor "google.golang.org/protobuf/types/descriptorpb"
)

func TestEditionsParser_ParseHTTPRules(t *testing.T) {
	t.Parallel()

	parser := NewEditionsParser()

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
			method: &descriptor.MethodDescriptorProto{
				Name:    proto.String("GetUser"),
				Options: &descriptor.MethodOptions{},
			},
			expected: []HTTPRule{},
		},
		{
			name: "method_with_multiple_rules",
			method: func() *descriptor.MethodDescriptorProto {
				method := &descriptor.MethodDescriptorProto{
					Name:    proto.String("UpdateUser"),
					Options: &descriptor.MethodOptions{},
				}

				// Create HTTP rule with additional bindings
				httpRule := &options.HttpRule{
					Pattern: &options.HttpRule_Put{
						Put: "/v1/users/{user_id}",
					},
					Body: "user",
					AdditionalBindings: []*options.HttpRule{
						{
							Pattern: &options.HttpRule_Patch{
								Patch: "/v1/users/{user_id}",
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
					Method:     "PUT",
					Pattern:    "/v1/users/{user_id}",
					Body:       "user",
					PathParams: []string{"user_id"},
				},
				{
					Method:     "PATCH",
					Pattern:    "/v1/users/{user_id}",
					Body:       "user",
					PathParams: []string{"user_id"},
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

func TestEditionsParser_ParsePathParams(t *testing.T) {
	t.Parallel()

	parser := NewEditionsParser()

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
			name:     "param_with_field_path",
			pattern:  "/v1/users/{user.id}",
			expected: []string{"user.id"},
		},
		{
			name:     "complex_params",
			pattern:  "/v1/organizations/{org_id}/users/{user_id}/settings/{setting_name}",
			expected: []string{"org_id", "user_id", "setting_name"},
		},
		{
			name:     "param_at_end",
			pattern:  "/v1/users/{id}",
			expected: []string{"id"},
		},
		{
			name:     "param_in_middle_and_end",
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

func TestEditionsParser_ConvertPathPattern(t *testing.T) {
	t.Parallel()

	parser := NewEditionsParser()

	tests := []struct {
		name     string
		pattern  string
		expected string
	}{
		{
			name:     "no_conversion_needed",
			pattern:  "/v1/users/{id}",
			expected: "/v1/users/{id}",
		},
		{
			name:     "simple_path",
			pattern:  "/api/test",
			expected: "/api/test",
		},
		{
			name:     "multiple_params",
			pattern:  "/v1/users/{user_id}/posts/{post_id}",
			expected: "/v1/users/{user_id}/posts/{post_id}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parser.ConvertPathPattern(tt.pattern)

			if result != tt.expected {
				t.Errorf("ConvertPathPattern(%q) = %q, want %q", tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestEditionsParser_parseHTTPRule(t *testing.T) {
	t.Parallel()

	parser := NewEditionsParser()

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
			name: "post_rule",
			httpRule: &options.HttpRule{
				Pattern: &options.HttpRule_Post{
					Post: "/v1/users",
				},
				Body: "user",
			},
			expected: HTTPRule{
				Method:     "POST",
				Pattern:    "/v1/users",
				Body:       "user",
				PathParams: []string{},
			},
		},
		{
			name: "put_rule",
			httpRule: &options.HttpRule{
				Pattern: &options.HttpRule_Put{
					Put: "/v1/users/{id}",
				},
				Body: "user",
			},
			expected: HTTPRule{
				Method:     "PUT",
				Pattern:    "/v1/users/{id}",
				Body:       "user",
				PathParams: []string{"id"},
			},
		},
		{
			name: "delete_rule",
			httpRule: &options.HttpRule{
				Pattern: &options.HttpRule_Delete{
					Delete: "/v1/users/{id}",
				},
				Body: "",
			},
			expected: HTTPRule{
				Method:     "DELETE",
				Pattern:    "/v1/users/{id}",
				Body:       "",
				PathParams: []string{"id"},
			},
		},
		{
			name: "patch_rule",
			httpRule: &options.HttpRule{
				Pattern: &options.HttpRule_Patch{
					Patch: "/v1/users/{id}",
				},
				Body: "user",
			},
			expected: HTTPRule{
				Method:     "PATCH",
				Pattern:    "/v1/users/{id}",
				Body:       "user",
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
