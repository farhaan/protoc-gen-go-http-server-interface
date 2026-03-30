package httpinterface_test

import (
	goparser "go/parser"
	"go/token"
	"testing"

	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface"
	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface/parser"
)

// FuzzGenerateCode tests GenerateCode with random service/method/pattern inputs.
// Properties checked:
//  1. GenerateCode never panics.
//  2. If GenerateCode returns no error, the output is valid Go syntax.
func FuzzGenerateCode(f *testing.F) {
	// Seed corpus: (packageName, serviceName, methodName, pattern, httpMethod)
	type seed struct {
		pkg, svc, method, pattern, httpMethod string
	}
	seeds := []seed{
		{"mypkg", "MyService", "GetItem", "/items/{id}", "GET"},
		{"mypkg", "MyService", "CreateItem", "/items", "POST"},
		{"apiv1", "UserService", "GetUser", "/users/{user_id}", "GET"},
		{"apiv1", "UserService", "UpdateUser", "/users/{id}", "PUT"},
		{"pkg", "S", "M", "/a/{b}/c/{d}", "PATCH"},
		{"pkg", "S", "Delete", "/x/{id}", "DELETE"},
		{"empty", "", "", "", "GET"},
		{"pkg", "S", "M", "/", "GET"},
		{"p", "A", "B", "/path/with/{nested}/{params}", "POST"},
		// Previously failing inputs — now handled correctly:
		{"A", "A", "A", "0", "0"},        // digit-only method (custom) — now strconv.Quote'd
		{"A", "A", "A", `"`, "GET"},      // quote in pattern — now goQuote'd
		{"A", "A", "A", `\path`, "GET"},  // backslash in pattern — now goQuote'd
	}

	for _, s := range seeds {
		f.Add(s.pkg, s.svc, s.method, s.pattern, s.httpMethod)
	}

	g := httpinterface.New()

	f.Fuzz(func(t *testing.T, pkgName, svcName, methodName, pattern, httpMethod string) {
		serviceData := &httpinterface.ServiceData{
			PackageName: pkgName,
			Services: []httpinterface.ServiceInfo{
				{
					Name: svcName,
					Methods: []httpinterface.MethodInfo{
						{
							Name: methodName,
							HTTPRules: []parser.HTTPRule{
								{Method: httpMethod, Pattern: pattern},
							},
						},
					},
				},
			},
		}

		// Property 1: must never panic (enforced by the fuzzer itself).
		generated, err := g.GenerateCode(serviceData)
		if err != nil {
			// Errors are acceptable for invalid Go identifiers or other malformed inputs.
			return
		}

		// Property 2: if no error, the output must be parseable Go.
		fset := token.NewFileSet()
		if _, parseErr := goparser.ParseFile(fset, "", generated, goparser.AllErrors); parseErr != nil {
			t.Errorf("GenerateCode produced unparseable Go for pkg=%q svc=%q method=%q pattern=%q httpMethod=%q:\n%v\n\nCode:\n%s",
				pkgName, svcName, methodName, pattern, httpMethod, parseErr, generated)
		}
	})
}

// FuzzGenerateCode_MultiMethod fuzzes a service with multiple methods.
func FuzzGenerateCode_MultiMethod(f *testing.F) {
	// Seed: (pkgName, m1Name, m1Pattern, m2Name, m2Pattern)
	f.Add("pkg", "GetA", "/a/{id}", "CreateB", "/b")
	f.Add("apiv1", "List", "/items", "Get", "/items/{id}")
	f.Add("svc", "Update", "/x/{id}", "Delete", "/x/{id}")
	// Patterns with characters that were previously unsafe in string literals:
	f.Add("pkg", "GetA", `"path"`, "CreateB", `path\n`)
	// Note: invalid UTF-8 bytes are not tested here — proto files are always valid UTF-8,
	// so GenerateCode is not designed to sanitize arbitrary byte sequences.

	g := httpinterface.New()

	f.Fuzz(func(t *testing.T, pkgName, m1Name, m1Pattern, m2Name, m2Pattern string) {
		if m1Name == m2Name {
			return // duplicate method names in same service are not meaningful
		}

		serviceData := &httpinterface.ServiceData{
			PackageName: pkgName,
			Services: []httpinterface.ServiceInfo{
				{
					Name: "FuzzService",
					Methods: []httpinterface.MethodInfo{
						{Name: m1Name, HTTPRules: []parser.HTTPRule{{Method: "GET", Pattern: m1Pattern}}},
						{Name: m2Name, HTTPRules: []parser.HTTPRule{{Method: "POST", Pattern: m2Pattern}}},
					},
				},
			},
		}

		generated, err := g.GenerateCode(serviceData)
		if err != nil {
			return
		}

		fset := token.NewFileSet()
		if _, parseErr := goparser.ParseFile(fset, "", generated, goparser.AllErrors); parseErr != nil {
			t.Errorf("multi-method fuzz: unparseable Go for pkg=%q, m1=%q/%q, m2=%q/%q:\n%v",
				pkgName, m1Name, m1Pattern, m2Name, m2Pattern, parseErr)
		}
	})
}
