package performance_test

import (
	"strings"
	"testing"

	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface"
	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface/parser"
)

// TestSecurity_MaliciousInputs tests handling of potentially malicious proto patterns
func TestSecurity_MaliciousInputs(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	tests := []struct {
		name        string
		serviceData *httpinterface.ServiceData
		shouldPanic bool
	}{
		{
			name: "extremely_long_service_name",
			serviceData: &httpinterface.ServiceData{
				PackageName: "security",
				Services: []httpinterface.ServiceInfo{
					{
						Name: strings.Repeat("A", 10000), // Very long name
						Methods: []httpinterface.MethodInfo{
							{
								Name:       "TestMethod",
								InputType:  "TestRequest",
								OutputType: "TestResponse",
								HTTPRules: []parser.HTTPRule{
									{Method: "GET", Pattern: "/test", Body: ""},
								},
							},
						},
					},
				},
			},
			shouldPanic: false, // Should handle gracefully
		},
		{
			name: "deeply_nested_path_parameters",
			serviceData: &httpinterface.ServiceData{
				PackageName: "security",
				Services: []httpinterface.ServiceInfo{
					{
						Name: "SecurityTestService",
						Methods: []httpinterface.MethodInfo{
							{
								Name:       "TestMethod",
								InputType:  "TestRequest",
								OutputType: "TestResponse",
								HTTPRules: []parser.HTTPRule{
									{
										Method:     "GET",
										Pattern:    "/a/{a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x.y.z}",
										Body:       "",
										PathParams: []string{"a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x.y.z"},
									},
								},
							},
						},
					},
				},
			},
			shouldPanic: false,
		},
		{
			name: "path_with_special_characters",
			serviceData: &httpinterface.ServiceData{
				PackageName: "security",
				Services: []httpinterface.ServiceInfo{
					{
						Name: "SecurityTestService",
						Methods: []httpinterface.MethodInfo{
							{
								Name:       "TestMethod",
								InputType:  "TestRequest",
								OutputType: "TestResponse",
								HTTPRules: []parser.HTTPRule{
									{
										Method:     "GET",
										Pattern:    "/test/{id}/../../../etc/passwd",
										Body:       "",
										PathParams: []string{"id"},
									},
								},
							},
						},
					},
				},
			},
			shouldPanic: false, // Should not cause security issues during generation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r != nil {
					if !tt.shouldPanic {
						t.Errorf("Test panicked unexpectedly: %v", r)
					}
				}
			}()

			generated, err := generator.GenerateCode(tt.serviceData)
			if err != nil && !tt.shouldPanic {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.shouldPanic && generated == "" {
				t.Error("Expected generated code but got empty string")
			}
		})
	}
}
