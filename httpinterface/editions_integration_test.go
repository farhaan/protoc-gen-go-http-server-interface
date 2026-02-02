package httpinterface

import (
	"net/http"
	"strings"
	"testing"

	options "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	descriptor "google.golang.org/protobuf/types/descriptorpb"
	plugin "google.golang.org/protobuf/types/pluginpb"
)

func TestGeneratorEditionsSupport(t *testing.T) {
	t.Parallel()

	g := New()

	// Test that the generator supports editions
	if !g.SupportsEditions {
		t.Error("Generator should support editions")
	}

	// Create a simple request
	req := &plugin.CodeGeneratorRequest{
		Parameter: proto.String("paths=source_relative"),
		ProtoFile: []*descriptor.FileDescriptorProto{
			{
				Name:    proto.String("test.proto"),
				Package: proto.String("test"),
				// Simulate an editions file
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
				Service: []*descriptor.ServiceDescriptorProto{
					{
						Name: proto.String("TestService"),
						Method: []*descriptor.MethodDescriptorProto{
							{
								Name:       proto.String("GetUser"),
								InputType:  proto.String(".test.GetUserRequest"),
								OutputType: proto.String(".test.User"),
								Options: func() *descriptor.MethodOptions {
									opts := &descriptor.MethodOptions{}
									httpRule := &options.HttpRule{
										Pattern: &options.HttpRule_Get{
											Get: "/v1/users/{id}",
										},
									}
									proto.SetExtension(opts, options.E_Http, httpRule)
									return opts
								}(),
							},
						},
					},
				},
				MessageType: []*descriptor.DescriptorProto{
					{Name: proto.String("GetUserRequest")},
					{Name: proto.String("User")},
				},
			},
		},
		FileToGenerate: []string{"test.proto"},
	}

	resp := g.Generate(req)

	// Check for errors
	if resp.Error != nil {
		t.Fatalf("Generation failed: %s", *resp.Error)
	}

	// Check supported features include editions
	if resp.SupportedFeatures == nil {
		t.Fatal("SupportedFeatures should be set")
	}

	features := *resp.SupportedFeatures
	if (features & uint64(plugin.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS)) == 0 {
		t.Error("SupportedFeatures should include FEATURE_SUPPORTS_EDITIONS")
	}

	// Check maximum edition is set
	if resp.MaximumEdition == nil {
		t.Error("MaximumEdition should be set for editions support")
	} else if *resp.MaximumEdition != int32(descriptor.Edition_EDITION_2023) {
		t.Errorf("MaximumEdition = %d, want %d", *resp.MaximumEdition, int32(descriptor.Edition_EDITION_2023))
	}

	// Check that files are generated
	if len(resp.File) != 1 {
		t.Errorf("len(resp.File) = %d, want 1", len(resp.File))
	}
}

func TestGeneratorWithEditionsParser(t *testing.T) {
	t.Parallel()

	// Create a file descriptor that triggers editions parser
	editionsFile := &descriptor.FileDescriptorProto{
		Name:    proto.String("editions_test.proto"),
		Package: proto.String("test"),
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
	}

	// Test that the correct extractors are created
	httpExtractor := CreateHTTPRuleExtractorForFile(editionsFile)
	pathExtractor := CreatePathParamExtractorForFile(editionsFile)
	patternConverter := CreatePathPatternConverterForFile(editionsFile)

	// Create generator with custom extractors
	g := NewWith(httpExtractor, pathExtractor, patternConverter)

	if !g.SupportsEditions {
		t.Error("Generator should support editions")
	}

	// Create method with HTTP rule
	method := &descriptor.MethodDescriptorProto{
		Name:       proto.String("UpdateUser"),
		InputType:  proto.String(".test.UpdateUserRequest"),
		OutputType: proto.String(".test.User"),
		Options: func() *descriptor.MethodOptions {
			opts := &descriptor.MethodOptions{}
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
			proto.SetExtension(opts, options.E_Http, httpRule)
			return opts
		}(),
	}

	// Test HTTP rule extraction
	rules := httpExtractor(method)
	if len(rules) != 2 {
		t.Errorf("len(rules) = %d, want 2", len(rules))
	}

	expectedRules := []struct {
		method  string
		pattern string
		body    string
	}{
		{"PUT", "/v1/users/{user_id}", "user"},
		{"PATCH", "/v1/users/{user_id}", "user"},
	}

	for i, expected := range expectedRules {
		if i >= len(rules) {
			continue
		}
		if rules[i].Method != expected.method {
			t.Errorf("rules[%d].Method = %q, want %q", i, rules[i].Method, expected.method)
		}
		if rules[i].Pattern != expected.pattern {
			t.Errorf("rules[%d].Pattern = %q, want %q", i, rules[i].Pattern, expected.pattern)
		}
		if rules[i].Body != expected.body {
			t.Errorf("rules[%d].Body = %q, want %q", i, rules[i].Body, expected.body)
		}
	}

	// Test path parameter extraction
	params := pathExtractor("/v1/users/{user_id}")
	expectedParams := []string{"user_id"}
	if len(params) != len(expectedParams) || params[0] != expectedParams[0] {
		t.Errorf("pathExtractor result = %v, want %v", params, expectedParams)
	}

	// Test pattern conversion
	converted := patternConverter("/v1/users/{user_id}")
	expected := "/v1/users/{user_id}"
	if converted != expected {
		t.Errorf("patternConverter result = %q, want %q", converted, expected)
	}
}

func TestEditionsFileDetection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		file             *descriptor.FileDescriptorProto
		shouldBeEditions bool
	}{
		{
			name: "proto2_file",
			file: &descriptor.FileDescriptorProto{
				Syntax: proto.String("proto2"),
			},
			shouldBeEditions: false,
		},
		{
			name: "proto3_file",
			file: &descriptor.FileDescriptorProto{
				Syntax: proto.String("proto3"),
			},
			shouldBeEditions: false,
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
			shouldBeEditions: true,
		},
		{
			name:             "empty_file",
			file:             &descriptor.FileDescriptorProto{},
			shouldBeEditions: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create extractor for the file
			extractor := CreateHTTPRuleExtractorForFile(tt.file)

			// Test with a simple method
			method := &descriptor.MethodDescriptorProto{
				Name:       proto.String("TestMethod"),
				InputType:  proto.String(".test.Request"),
				OutputType: proto.String(".test.Response"),
				Options: func() *descriptor.MethodOptions {
					opts := &descriptor.MethodOptions{}
					httpRule := &options.HttpRule{
						Pattern: &options.HttpRule_Get{
							Get: "/v1/test/{id}",
						},
					}
					proto.SetExtension(opts, options.E_Http, httpRule)
					return opts
				}(),
			}

			rules := extractor(method)

			// All parsers should extract the rule correctly
			if len(rules) != 1 {
				t.Errorf("len(rules) = %d, want 1", len(rules))
				return
			}

			if rules[0].Method != http.MethodGet {
				t.Errorf("rules[0].Method = %q, want %q", rules[0].Method, http.MethodGet)
			}

			if rules[0].Pattern != "/v1/test/{id}" {
				t.Errorf("rules[0].Pattern = %q, want %q", rules[0].Pattern, "/v1/test/{id}")
			}

			// Verify path params are extracted
			if len(rules[0].PathParams) != 1 || rules[0].PathParams[0] != "id" {
				t.Errorf("rules[0].PathParams = %v, want [\"id\"]", rules[0].PathParams)
			}
		})
	}
}

func TestCompleteEditionsWorkflow(t *testing.T) {
	t.Parallel()

	// Create a complete editions proto file
	req := &plugin.CodeGeneratorRequest{
		Parameter: proto.String("paths=source_relative"),
		ProtoFile: []*descriptor.FileDescriptorProto{
			{
				Name:    proto.String("complete_test.proto"),
				Package: proto.String("complete"),
				// Editions file
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
				Service: []*descriptor.ServiceDescriptorProto{
					{
						Name: proto.String("UserService"),
						Method: []*descriptor.MethodDescriptorProto{
							{
								Name:       proto.String("GetUser"),
								InputType:  proto.String(".complete.GetUserRequest"),
								OutputType: proto.String(".complete.User"),
								Options: func() *descriptor.MethodOptions {
									opts := &descriptor.MethodOptions{}
									httpRule := &options.HttpRule{
										Pattern: &options.HttpRule_Get{
											Get: "/v1/users/{id}",
										},
									}
									proto.SetExtension(opts, options.E_Http, httpRule)
									return opts
								}(),
							},
							{
								Name:       proto.String("CreateUser"),
								InputType:  proto.String(".complete.CreateUserRequest"),
								OutputType: proto.String(".complete.User"),
								Options: func() *descriptor.MethodOptions {
									opts := &descriptor.MethodOptions{}
									httpRule := &options.HttpRule{
										Pattern: &options.HttpRule_Post{
											Post: "/v1/users",
										},
										Body: "user",
									}
									proto.SetExtension(opts, options.E_Http, httpRule)
									return opts
								}(),
							},
							{
								Name:       proto.String("UpdateUser"),
								InputType:  proto.String(".complete.UpdateUserRequest"),
								OutputType: proto.String(".complete.User"),
								Options: func() *descriptor.MethodOptions {
									opts := &descriptor.MethodOptions{}
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
											{
												Pattern: &options.HttpRule_Custom{
													Custom: &options.CustomHttpPattern{
														Kind: "MERGE",
														Path: "/v1/users/{user_id}/merge",
													},
												},
												Body: "user",
											},
										},
									}
									proto.SetExtension(opts, options.E_Http, httpRule)
									return opts
								}(),
							},
						},
					},
				},
				MessageType: []*descriptor.DescriptorProto{
					{Name: proto.String("GetUserRequest")},
					{Name: proto.String("CreateUserRequest")},
					{Name: proto.String("UpdateUserRequest")},
					{Name: proto.String("User")},
				},
			},
		},
		FileToGenerate: []string{"complete_test.proto"},
	}

	g := New()
	resp := g.Generate(req)

	// Check for errors
	if resp.Error != nil {
		t.Fatalf("Generation failed: %s", *resp.Error)
	}

	// Verify editions support is declared
	if resp.SupportedFeatures == nil {
		t.Fatal("SupportedFeatures should be set")
	}

	features := *resp.SupportedFeatures
	if (features & uint64(plugin.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS)) == 0 {
		t.Error("SupportedFeatures should include FEATURE_SUPPORTS_EDITIONS")
	}

	// Check that file is generated
	if len(resp.File) != 1 {
		t.Fatalf("len(resp.File) = %d, want 1", len(resp.File))
	}

	generatedFile := resp.File[0]
	if generatedFile.Name == nil {
		t.Fatal("Generated file name is nil")
	}

	expectedFilename := "complete_test_http.pb.go"
	if *generatedFile.Name != expectedFilename {
		t.Errorf("Generated file name = %q, want %q", *generatedFile.Name, expectedFilename)
	}

	// Check content contains expected elements
	content := *generatedFile.Content

	expectedContent := []string{
		"package complete",
		"type UserServiceHandler interface",
		"HandleGetUser(w http.ResponseWriter, r *http.Request)",
		"HandleCreateUser(w http.ResponseWriter, r *http.Request)",
		"HandleUpdateUser(w http.ResponseWriter, r *http.Request)",
		`r.HandleFunc(http.MethodGet, "/v1/users/{id}", handler.HandleGetUser)`,
		`r.HandleFunc(http.MethodPost, "/v1/users", handler.HandleCreateUser)`,
		`r.HandleFunc(http.MethodPut, "/v1/users/{user_id}", handler.HandleUpdateUser)`,
		`r.HandleFunc(http.MethodPatch, "/v1/users/{user_id}", handler.HandleUpdateUser)`,
		`r.HandleFunc("MERGE", "/v1/users/{user_id}/merge", handler.HandleUpdateUser)`,
	}

	for _, expected := range expectedContent {
		if !strings.Contains(content, expected) {
			t.Errorf("Generated content missing expected part: %q", expected)
		}
	}
}
