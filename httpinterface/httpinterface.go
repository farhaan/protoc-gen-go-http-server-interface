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
	// Options contains the plugin options
	Options *Options
	// SupportsEditions indicates if this generator supports editions
	SupportsEditions bool
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
		ParsedTemplates:  tmpl,
		Options:          &Options{},
		SupportsEditions: true,
	}
}

// Generate generates the HTTP interface code.
func (g *Generator) Generate(req *plugin.CodeGeneratorRequest) *plugin.CodeGeneratorResponse {
	resp := new(plugin.CodeGeneratorResponse)

	// resp.MinimumEdition = proto.String("proto2")

	// Parse options from parameter
	options, err := ParseOptions(req.GetParameter())
	if err != nil {
		resp.Error = proto.String(fmt.Sprintf("invalid options: %v", err))
		return resp
	}
	g.Options = options
	if options.SupportsEditions {
		g.SupportsEditions = true
	}

	// Advertise editions support in the response
	supportedFeatures := uint64(plugin.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
	if g.SupportsEditions {
		supportedFeatures |= uint64(plugin.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS)
		minimumEdition := int32(plugin.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		maximumEdition := int32(plugin.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		resp.MinimumEdition = &minimumEdition
		resp.MaximumEdition = &maximumEdition
	}
	resp.SupportedFeatures = proto.Uint64(supportedFeatures)

	// Process each proto file
	for _, file := range req.ProtoFile {
		if !g.shouldGenerate(file.GetName(), req.FileToGenerate) {
			continue
		}

		// Set the current file descriptor for processing
		SetFileDescriptor(file)

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
		outputFile := &plugin.CodeGeneratorResponse_File{
			Name:    proto.String(filename),
			Content: proto.String(content),
		}

		// Handle source_relative paths option
		if g.Options.PathsSourceRelative {
			// Get directory of the proto file
			dir := filepath.Dir(file.GetName())
			if dir != "." {
				outputFile.Name = proto.String(filepath.Join(dir, filename))
			}
		}

		resp.File = append(resp.File, outputFile)
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
			for i, rule := range httpRules {
				// Create a copy of the rule
				methodInfo.HTTPRules[i] = HTTPRule{
					Method:     rule.Method,
					Pattern:    rule.Pattern,
					Body:       rule.Body,
					PathParams: rule.PathParams,
				}

				// Apply pattern conversion
				methodInfo.HTTPRules[i].Pattern = ConvertPathPattern(methodInfo.HTTPRules[i].Pattern)

				// CRITICAL FIX: If PathParams is empty, populate it explicitly, even when using mocks
				if len(methodInfo.HTTPRules[i].PathParams) == 0 {
					methodInfo.HTTPRules[i].PathParams = GetPathParams(methodInfo.HTTPRules[i].Pattern)
				}
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
	filename := strings.TrimSuffix(base, ".proto")

	if g.Options.OutputPrefix != "" {
		filename = g.Options.OutputPrefix + "_" + filename
	} else {
		filename = filename + "_http"
	}

	return filename + ".pb.go"
}

// getPackageName returns the Go package name for a proto file.
func (g *Generator) getPackageName(file *descriptor.FileDescriptorProto) string {
	// Use go_package option if available
	if goPackage := file.GetOptions().GetGoPackage(); goPackage != "" {
		// Check for explicit package name after semicolon
		if idx := strings.LastIndex(goPackage, ";"); idx >= 0 {
			return goPackage[idx+1:]
		}
		// Otherwise use the last path segment
		if idx := strings.LastIndex(goPackage, "/"); idx >= 0 {
			return goPackage[idx+1:]
		}
		return goPackage
	}

	// Fall back to proto package name
	protoPackage := file.GetPackage()
	if protoPackage == "" {
		return ""
	}

	// Split by dots and process
	parts := strings.Split(protoPackage, ".")
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return strings.Join(parts, "")
	default:
		// For packages with more than 2 segments, use the last two
		// Example: api.core.oauth.v1 -> oauthv1
		lastTwo := parts[len(parts)-2:]
		return strings.Join(lastTwo, "")
	}
}

// getTypeName returns the simple type name from a fully qualified type name.
func (g *Generator) getTypeName(typeName string) string {
	parts := strings.Split(typeName, ".")
	return parts[len(parts)-1]
}
