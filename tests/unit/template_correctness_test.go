package unit_test

import (
	goparser "go/parser"
	"go/format"
	"go/token"
	"strings"
	"testing"

	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface"
	parserlib "github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface/parser"
)

// T18: Syntax validity — generated code must parse without errors.
func TestTemplate_ValidGoSyntax(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	testCases := []struct {
		name        string
		serviceData *httpinterface.ServiceData
	}{
		{
			name: "basic_service",
			serviceData: &httpinterface.ServiceData{
				PackageName: "testpkg",
				Services: []httpinterface.ServiceInfo{
					{
						Name: "TestService",
						Methods: []httpinterface.MethodInfo{
							{
								Name: "GetItem",
								HTTPRules: []parserlib.HTTPRule{
									{Method: "GET", Pattern: "/items/{id}", PathParams: []string{"id"}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "multi_binding_method",
			serviceData: &httpinterface.ServiceData{
				PackageName: "testpkg",
				Services: []httpinterface.ServiceInfo{
					{
						Name: "UpdateService",
						Methods: []httpinterface.MethodInfo{
							{
								Name: "Update",
								HTTPRules: []parserlib.HTTPRule{
									{Method: "PUT", Pattern: "/items/{id}", PathParams: []string{"id"}},
									{Method: "PATCH", Pattern: "/items/{id}", PathParams: []string{"id"}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "no_path_params",
			serviceData: &httpinterface.ServiceData{
				PackageName: "testpkg",
				Services: []httpinterface.ServiceInfo{
					{
						Name: "PingService",
						Methods: []httpinterface.MethodInfo{
							{
								Name:      "Ping",
								HTTPRules: []parserlib.HTTPRule{{Method: "GET", Pattern: "/ping"}},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple_services",
			serviceData: &httpinterface.ServiceData{
				PackageName: "testpkg",
				Services: []httpinterface.ServiceInfo{
					{
						Name: "ServiceA",
						Methods: []httpinterface.MethodInfo{
							{Name: "MethodA", HTTPRules: []parserlib.HTTPRule{{Method: "GET", Pattern: "/a"}}},
						},
					},
					{
						Name: "ServiceB",
						Methods: []httpinterface.MethodInfo{
							{Name: "MethodB", HTTPRules: []parserlib.HTTPRule{{Method: "POST", Pattern: "/b", Body: "*"}}},
						},
					},
				},
			},
		},
		{
			name: "many_path_params",
			serviceData: &httpinterface.ServiceData{
				PackageName: "testpkg",
				Services: []httpinterface.ServiceInfo{
					{
						Name: "AssignService",
						Methods: []httpinterface.MethodInfo{
							{
								Name: "Assign",
								HTTPRules: []parserlib.HTTPRule{{
									Method:     "POST",
									Pattern:    "/orgs/{org_id}/projects/{project_id}/tasks/{task_id}/assign/{user_id}",
									PathParams: []string{"org_id", "project_id", "task_id", "user_id"},
								}},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			generated, err := generator.GenerateCode(tc.serviceData)
			if err != nil {
				t.Fatalf("GenerateCode: %v", err)
			}

			fset := token.NewFileSet()
			_, err = goparser.ParseFile(fset, "", generated, goparser.AllErrors)
			if err != nil {
				t.Errorf("generated code is not valid Go:\n%v\n\nCode:\n%s", err, generated)
			}
		})
	}
}

// T19: gofmt compliance — generated code must be gofmt-formatted already.
func TestTemplate_GofmtCompliance(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	serviceData := &httpinterface.ServiceData{
		PackageName: "fmtpkg",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "FmtService",
				Methods: []httpinterface.MethodInfo{
					{
						Name: "GetItem",
						HTTPRules: []parserlib.HTTPRule{
							{Method: "GET", Pattern: "/items/{id}", PathParams: []string{"id"}},
						},
					},
					{
						Name:      "Create",
						HTTPRules: []parserlib.HTTPRule{{Method: "POST", Pattern: "/items", Body: "*"}},
					},
				},
			},
		},
	}

	generated, err := generator.GenerateCode(serviceData)
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}

	formatted, err := format.Source([]byte(generated))
	if err != nil {
		t.Fatalf("format.Source error: %v\n\nCode:\n%s", err, generated)
	}

	if string(formatted) != generated {
		// Compute a simple diff by finding the first differing line.
		wantLines := strings.Split(string(formatted), "\n")
		gotLines := strings.Split(generated, "\n")
		for i := 0; i < len(wantLines) && i < len(gotLines); i++ {
			if wantLines[i] != gotLines[i] {
				t.Errorf("gofmt difference at line %d:\n  want: %q\n   got: %q", i+1, wantLines[i], gotLines[i])
				break
			}
		}
		if !t.Failed() {
			t.Errorf("generated code differs from gofmt output (different lengths: want %d lines, got %d)",
				len(wantLines), len(gotLines))
		}
	}
}

// T20: Determinism — generating the same input twice produces identical output.
func TestTemplate_Determinism(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	serviceData := &httpinterface.ServiceData{
		PackageName: "detpkg",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "DetService",
				Methods: []httpinterface.MethodInfo{
					{Name: "MethodA", HTTPRules: []parserlib.HTTPRule{{Method: "GET", Pattern: "/a"}}},
					{Name: "MethodB", HTTPRules: []parserlib.HTTPRule{{Method: "POST", Pattern: "/b", Body: "*"}}},
					{Name: "MethodC", HTTPRules: []parserlib.HTTPRule{
						{Method: "PUT", Pattern: "/c/{id}", PathParams: []string{"id"}},
						{Method: "PATCH", Pattern: "/c/{id}", PathParams: []string{"id"}},
					}},
				},
			},
		},
	}

	first, err := generator.GenerateCode(serviceData)
	if err != nil {
		t.Fatalf("first GenerateCode: %v", err)
	}

	second, err := generator.GenerateCode(serviceData)
	if err != nil {
		t.Fatalf("second GenerateCode: %v", err)
	}

	if first != second {
		t.Error("GenerateCode is not deterministic: two calls with same input gave different output")
	}
}

// T21: Template injection safety — service/method names with special chars don't break Go syntax.
func TestTemplate_InjectionSafety(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	// These are valid Go identifiers but potentially tricky names.
	testCases := []struct {
		name        string
		serviceName string
		methodName  string
	}{
		{name: "numbers_in_name", serviceName: "Service2", methodName: "GetV2"},
		{name: "underscore_in_name", serviceName: "My_Service", methodName: "Get_Item"},
		{name: "all_caps", serviceName: "APIService", methodName: "GetHTTPItem"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			serviceData := &httpinterface.ServiceData{
				PackageName: "safepkg",
				Services: []httpinterface.ServiceInfo{
					{
						Name: tc.serviceName,
						Methods: []httpinterface.MethodInfo{
							{
								Name:      tc.methodName,
								HTTPRules: []parserlib.HTTPRule{{Method: "GET", Pattern: "/test"}},
							},
						},
					},
				},
			}

			generated, err := generator.GenerateCode(serviceData)
			if err != nil {
				t.Fatalf("GenerateCode: %v", err)
			}

			fset := token.NewFileSet()
			_, parseErr := goparser.ParseFile(fset, "", generated, goparser.AllErrors)
			if parseErr != nil {
				t.Errorf("generated code for %q/%q is not valid Go: %v", tc.serviceName, tc.methodName, parseErr)
			}
		})
	}
}

// T22: Interface/Unimplemented parity — every interface method has a corresponding Unimplemented stub.
func TestTemplate_InterfaceUnimplementedParity(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	methods := []httpinterface.MethodInfo{
		{Name: "GetFoo", HTTPRules: []parserlib.HTTPRule{{Method: "GET", Pattern: "/foo"}}},
		{Name: "CreateFoo", HTTPRules: []parserlib.HTTPRule{{Method: "POST", Pattern: "/foo", Body: "*"}}},
		{Name: "UpdateFoo", HTTPRules: []parserlib.HTTPRule{
			{Method: "PUT", Pattern: "/foo/{id}", PathParams: []string{"id"}},
			{Method: "PATCH", Pattern: "/foo/{id}", PathParams: []string{"id"}},
		}},
		{Name: "DeleteFoo", HTTPRules: []parserlib.HTTPRule{{Method: "DELETE", Pattern: "/foo/{id}", PathParams: []string{"id"}}}},
	}

	serviceData := &httpinterface.ServiceData{
		PackageName: "paritypkg",
		Services: []httpinterface.ServiceInfo{
			{Name: "FooService", Methods: methods},
		},
	}

	generated, err := generator.GenerateCode(serviceData)
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}

	for _, m := range methods {
		// Interface method declaration
		ifaceMethod := "Handle" + m.Name + "(w http.ResponseWriter, r *http.Request)"
		if !strings.Contains(generated, ifaceMethod) {
			t.Errorf("interface missing method: %s", ifaceMethod)
		}

		// Unimplemented stub
		stub := "func (UnimplementedFooServiceHandler) Handle" + m.Name
		if !strings.Contains(generated, stub) {
			t.Errorf("unimplemented stub missing: %s", stub)
		}

		// Stub returns 501 with the method name
		stubBody := `"not implemented: ` + m.Name + `"`
		if !strings.Contains(generated, stubBody) {
			t.Errorf("stub body missing for %s: %s", m.Name, stubBody)
		}
	}
}

// T23: Path param comment accuracy — only methods with path params have PathValue comments.
func TestTemplate_PathParamCommentAccuracy(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	serviceData := &httpinterface.ServiceData{
		PackageName: "commentpkg",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "CommentService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:      "WithParam",
						HTTPRules: []parserlib.HTTPRule{{Method: "GET", Pattern: "/items/{item_id}", PathParams: []string{"item_id"}}},
					},
					{
						Name:      "WithoutParam",
						HTTPRules: []parserlib.HTTPRule{{Method: "GET", Pattern: "/items"}},
					},
					{
						Name: "WithMultipleParams",
						HTTPRules: []parserlib.HTTPRule{{
							Method:     "POST",
							Pattern:    "/a/{p1}/b/{p2}/c/{p3}",
							PathParams: []string{"p1", "p2", "p3"},
						}},
					},
				},
			},
		},
	}

	generated, err := generator.GenerateCode(serviceData)
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}

	// WithParam — should have PathValue comment for item_id.
	if !strings.Contains(generated, `r.PathValue("item_id")`) {
		t.Error("WithParam: missing r.PathValue(\"item_id\")")
	}

	// WithoutParam — should NOT have any PathValue comment.
	// We look for the absence of PathValue in the vicinity of WithoutParam.
	// Simple check: count PathValue occurrences — should be only for methods with params.
	if strings.Count(generated, `// Path params`) != 2 {
		t.Errorf("expected 2 path-param comments (WithParam + WithMultipleParams), got %d",
			strings.Count(generated, `// Path params`))
	}

	// WithMultipleParams — should list all three params on one line.
	wantMulti := `r.PathValue("p1"), r.PathValue("p2"), r.PathValue("p3")`
	if !strings.Contains(generated, wantMulti) {
		t.Errorf("WithMultipleParams: missing multi-param line %q", wantMulti)
	}

	// Test hint should appear for methods with path params.
	if strings.Count(generated, "In tests: use req.SetPathValue") != 2 {
		t.Errorf("expected 2 test hint comments, got %d",
			strings.Count(generated, "In tests: use req.SetPathValue"))
	}
}

// T24: All bindings registered — RegisterXxxRoutes must call HandleFunc for every HTTP rule.
func TestTemplate_AllBindingsRegistered(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	serviceData := &httpinterface.ServiceData{
		PackageName: "bindpkg",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "BindService",
				Methods: []httpinterface.MethodInfo{
					{
						Name: "Update",
						HTTPRules: []parserlib.HTTPRule{
							{Method: "PUT", Pattern: "/items/{id}", PathParams: []string{"id"}},
							{Method: "PATCH", Pattern: "/items/{id}", PathParams: []string{"id"}},
						},
					},
					{
						Name:      "Create",
						HTTPRules: []parserlib.HTTPRule{{Method: "POST", Pattern: "/items", Body: "*"}},
					},
					{
						Name:      "Delete",
						HTTPRules: []parserlib.HTTPRule{{Method: "DELETE", Pattern: "/items/{id}", PathParams: []string{"id"}}},
					},
				},
			},
		},
	}

	generated, err := generator.GenerateCode(serviceData)
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}

	// Every HTTP rule must appear in the registration function.
	expectations := []string{
		`http.MethodPut, "/items/{id}"`,
		`http.MethodPatch, "/items/{id}"`,
		`http.MethodPost, "/items"`,
		`http.MethodDelete, "/items/{id}"`,
	}
	for _, want := range expectations {
		if !strings.Contains(generated, want) {
			t.Errorf("RegisterBindServiceRoutes missing binding: %s", want)
		}
	}

	// Multi-binding method should have correct binding count comment.
	if !strings.Contains(generated, "2 binding(s)") {
		t.Error("expected '2 binding(s)' comment for Update method with 2 bindings")
	}
}
