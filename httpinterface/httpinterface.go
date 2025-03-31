// Package httpinterface implements HTTP interface generation for protocol buffers.
package httpinterface

import (
	"bytes"
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"google.golang.org/protobuf/proto"
)

// Templates
var (
	//go:embed templates/header-template.go.tmpl
	headerTemplate string
	//go:embed templates/service-template.go.tmpl
	serviceTemplate string
)

// Generator is the httpinterface code generator.
type Generator struct {
	// ParsedTemplates contains the parsed templates for code generation
	ParsedTemplates *template.Template
}

// ServiceData contains the data for a service definition.
type ServiceData struct {
	PackageName string
	Services    []ServiceInfo
}

// ServiceInfo contains information about a service.
type ServiceInfo struct {
	Name    string
	Methods []MethodInfo
}

// MethodInfo contains information about a method.
type MethodInfo struct {
	Name       string
	InputType  string
	OutputType string
	HTTPRules  []HTTPRule
}

// HTTPRule represents an HTTP binding from annotations.
type HTTPRule struct {
	Method     string
	Pattern    string
	Body       string
	PathParams []string
}

// New creates a new httpinterface generator.
func New() *Generator {
	// Parse the templates
	tmpl := template.New("httpinterface").Funcs(template.FuncMap{
		"lower": strings.ToLower,
	})

	// Parse header template
	tmpl = template.Must(tmpl.New("header").Parse(headerTemplate))

	// Parse service template
	tmpl = template.Must(tmpl.New("service").Parse(serviceTemplate))

	return &Generator{
		ParsedTemplates: tmpl,
	}
}

// Generate generates the HTTP interface code.
func (g *Generator) Generate(req *plugin.CodeGeneratorRequest) *plugin.CodeGeneratorResponse {
	resp := new(plugin.CodeGeneratorResponse)

	// Process each proto file
	for _, file := range req.ProtoFile {
		if !g.shouldGenerate(file.GetName(), req.FileToGenerate) {
			continue
		}

		// Check if the file has any services with HTTP annotations
		if !g.hasHTTPRules(file) {
			continue
		}

		// Prepare the data for code generation
		data := g.buildServiceData(file)
		if len(data.Services) == 0 {
			continue
		}

		// Generate code
		content, err := g.generateCode(data)
		if err != nil {
			resp.Error = proto.String(fmt.Sprintf("error generating code for %s: %v", file.GetName(), err))
			return resp
		}

		// Add the file to the response
		filename := g.outputFilename(file.GetName())
		resp.File = append(resp.File, &plugin.CodeGeneratorResponse_File{
			Name:    proto.String(filename),
			Content: proto.String(content),
		})
	}

	return resp
}

// shouldGenerate returns whether code should be generated for the given file.
func (g *Generator) shouldGenerate(file string, filesToGenerate []string) bool {
	for _, f := range filesToGenerate {
		if f == file {
			return true
		}
	}
	return false
}

// hasHTTPRules returns whether the file has any services with HTTP annotations.
func (g *Generator) hasHTTPRules(file *descriptor.FileDescriptorProto) bool {
	for _, service := range file.Service {
		for _, method := range service.Method {
			rules := GetHTTPRules(method)
			if len(rules) > 0 {
				return true
			}
		}
	}
	return false
}

// buildServiceData builds the service data for code generation.
func (g *Generator) buildServiceData(file *descriptor.FileDescriptorProto) *ServiceData {
	data := &ServiceData{
		PackageName: g.getPackageName(file),
		Services:    make([]ServiceInfo, 0, len(file.Service)),
	}

	for _, service := range file.Service {
		serviceInfo := ServiceInfo{
			Name:    service.GetName(),
			Methods: make([]MethodInfo, 0, len(service.Method)),
		}

		for _, method := range service.Method {
			httpRules := GetHTTPRules(method)
			if len(httpRules) == 0 {
				continue
			}

			methodInfo := MethodInfo{
				Name:       method.GetName(),
				InputType:  g.getTypeName(method.GetInputType()),
				OutputType: g.getTypeName(method.GetOutputType()),
				HTTPRules:  httpRules,
			}

			// Process HTTP rules
			for i := range methodInfo.HTTPRules {
				rule := &methodInfo.HTTPRules[i]
				rule.PathParams = GetPathParams(rule.Pattern)
				rule.Pattern = ConvertPathPattern(rule.Pattern)
			}

			serviceInfo.Methods = append(serviceInfo.Methods, methodInfo)
		}

		if len(serviceInfo.Methods) > 0 {
			data.Services = append(data.Services, serviceInfo)
		}
	}

	return data
}

// generateCode generates the code from templates.
func (g *Generator) generateCode(data *ServiceData) (string, error) {
	var buf bytes.Buffer

	// Execute header template
	if err := g.ParsedTemplates.ExecuteTemplate(&buf, "header", data); err != nil {
		return "", fmt.Errorf("failed to execute header template: %v", err)
	}

	// Execute service template for each service
	for _, service := range data.Services {
		if err := g.ParsedTemplates.ExecuteTemplate(&buf, "service", service); err != nil {
			return "", fmt.Errorf("failed to execute service template for %s: %v", service.Name, err)
		}
	}

	return buf.String(), nil
}

// outputFilename returns the output filename for a proto file.
func (g *Generator) outputFilename(protoFilename string) string {
	base := filepath.Base(protoFilename)
	return strings.TrimSuffix(base, ".proto") + "_http.pb.go"
}

// getPackageName returns the Go package name for a proto file.
func (g *Generator) getPackageName(file *descriptor.FileDescriptorProto) string {
	// Use go_package option if available
	if goPackage := file.GetOptions().GetGoPackage(); goPackage != "" {
		if idx := strings.LastIndex(goPackage, "/"); idx >= 0 {
			return goPackage[idx+1:]
		}
		if idx := strings.LastIndex(goPackage, ";"); idx >= 0 {
			return goPackage[idx+1:]
		}
		return goPackage
	}

	// Fall back to proto package
	return file.GetPackage()
}

// getTypeName returns the simple type name from a fully qualified type name.
func (g *Generator) getTypeName(typeName string) string {
	parts := strings.Split(typeName, ".")
	return parts[len(parts)-1]
}
