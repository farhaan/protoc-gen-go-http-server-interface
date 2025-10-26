// Package httpinterface implements HTTP interface generation for protocol buffers.
package httpinterface

import (
	"bytes"
	_ "embed"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"google.golang.org/protobuf/proto"
	descriptor "google.golang.org/protobuf/types/descriptorpb"
	plugin "google.golang.org/protobuf/types/pluginpb"
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
	// HTTPRuleExtractor extracts HTTP rules from method descriptors
	HTTPRuleExtractor HTTPRuleExtractor
	// PathParamExtractor extracts path parameters from patterns
	PathParamExtractor PathParamExtractor
	// PathPatternConverter converts path patterns
	PathPatternConverter PathPatternConverter
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

// New creates a new httpinterface generator with an optional custom HTTP rule extractor.
// If no extractor is provided, uses the default parseMethodHTTPRules.
func New(httpExtractor ...HTTPRuleExtractor) *Generator {
	// Parse the templates
	tmpl := template.New("httpinterface").Funcs(template.FuncMap{
		"lower": strings.ToLower,
		"title": func(s string) string {
			if s == "" {
				return ""
			}
			return strings.ToUpper(s[:1]) + s[1:]
		},
	})

	// Parse header template
	tmpl = template.Must(tmpl.New("header").Parse(headerTemplate))

	// Parse service template
	tmpl = template.Must(tmpl.New("service").Parse(serviceTemplate))

	// Set up defaults
	var extractor HTTPRuleExtractor = parseMethodHTTPRules
	if len(httpExtractor) > 0 {
		extractor = httpExtractor[0]
	}

	return &Generator{
		ParsedTemplates:      tmpl,
		Options:              &Options{},
		HTTPRuleExtractor:    extractor,
		PathParamExtractor:   parsePathParams,
		PathPatternConverter: convertPathPattern,
		SupportsEditions:     true,
	}
}

// NewWith creates a new generator with all custom dependencies.
func NewWith(httpExtractor HTTPRuleExtractor, pathExtractor PathParamExtractor,
	converter PathPatternConverter) *Generator {
	// Parse the templates
	tmpl := template.New("httpinterface").Funcs(template.FuncMap{
		"lower": strings.ToLower,
		"title": func(s string) string {
			if s == "" {
				return ""
			}
			return strings.ToUpper(s[:1]) + s[1:]
		},
	})

	// Parse header template
	tmpl = template.Must(tmpl.New("header").Parse(headerTemplate))

	// Parse service template
	tmpl = template.Must(tmpl.New("service").Parse(serviceTemplate))

	return &Generator{
		ParsedTemplates:      tmpl,
		Options:              &Options{},
		HTTPRuleExtractor:    httpExtractor,
		PathParamExtractor:   pathExtractor,
		PathPatternConverter: converter,
		SupportsEditions:     true,
	}
}

// Generate generates the HTTP interface code.
func (g *Generator) Generate(req *plugin.CodeGeneratorRequest) *plugin.CodeGeneratorResponse {
	resp := new(plugin.CodeGeneratorResponse)

	// Parse options from parameter first
	if err := g.parseAndSetOptions(req.GetParameter()); err != nil {
		resp.Error = proto.String(fmt.Sprintf("invalid options: %v", err))
		return resp
	}

	// Set SupportsEditions based on options
	if g.Options != nil && g.Options.Editions {
		g.SupportsEditions = true
	}

	// Declare support for protobuf features
	supportedFeatures := uint64(plugin.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)

	// Add editions support if the generator supports it
	if g.SupportsEditions {
		supportedFeatures |= uint64(plugin.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS)
	}

	resp.SupportedFeatures = proto.Uint64(supportedFeatures)

	// Set edition support range for editions
	if g.SupportsEditions {
		resp.MinimumEdition = proto.Int32(int32(descriptor.Edition_EDITION_PROTO3))
		resp.MaximumEdition = proto.Int32(int32(descriptor.Edition_EDITION_2023))
	}

	// Process each proto file
	for _, file := range req.ProtoFile {
		if outputFile, err := g.processFile(file, req.FileToGenerate); err != nil {
			resp.Error = proto.String(err.Error())
			return resp
		} else if outputFile != nil {
			resp.File = append(resp.File, outputFile)
		}
	}

	return resp
}

// parseAndSetOptions parses options and sets them on the generator
func (g *Generator) parseAndSetOptions(parameter string) error {
	options, err := ParseOptions(parameter)
	if err != nil {
		return err
	}
	g.Options = options
	return nil
}

// processFile processes a single proto file and returns an output file if generation is needed
func (g *Generator) processFile(
	file *descriptor.FileDescriptorProto,
	filesToGenerate []string,
) (*plugin.CodeGeneratorResponse_File, error) {
	if !g.shouldGenerate(file.GetName(), filesToGenerate) {
		return nil, nil
	}

	// Check if the file has any services with HTTP annotations
	if !g.hasHTTPRules(file) {
		return nil, nil
	}

	// Prepare the data for code generation
	data := g.buildServiceData(file)
	if len(data.Services) == 0 {
		return nil, nil
	}

	// Generate code
	content, err := g.GenerateCode(data)
	if err != nil {
		return nil, fmt.Errorf("error generating code for %s: %v", file.GetName(), err)
	}

	// Create output file
	filename := g.outputFilename(file.GetName())
	outputFile := &plugin.CodeGeneratorResponse_File{
		Name:    proto.String(filename),
		Content: proto.String(content),
	}

	// Handle source_relative paths option
	g.adjustFilenameForSourceRelative(outputFile, file.GetName())

	return outputFile, nil
}

// adjustFilenameForSourceRelative adjusts the output filename for source_relative paths
func (g *Generator) adjustFilenameForSourceRelative(
	outputFile *plugin.CodeGeneratorResponse_File,
	protoFileName string,
) {
	if g.Options.PathsSourceRelative {
		dir := filepath.Dir(protoFileName)
		if dir != "." {
			filename := filepath.Base(outputFile.GetName())
			outputFile.Name = proto.String(filepath.Join(dir, filename))
		}
	}
}

// shouldGenerate returns whether code should be generated for the given file.
func (g *Generator) shouldGenerate(file string, filesToGenerate []string) bool {
	return slices.Contains(filesToGenerate, file)
}

// hasHTTPRules returns whether the file has any services with HTTP annotations.
func (g *Generator) hasHTTPRules(file *descriptor.FileDescriptorProto) bool {
	for _, service := range file.Service {
		for _, method := range service.Method {
			rules := g.HTTPRuleExtractor(method)
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
			httpRules := g.HTTPRuleExtractor(method)
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
				rule.PathParams = g.PathParamExtractor(rule.Pattern)
				rule.Pattern = g.PathPatternConverter(rule.Pattern)
			}

			serviceInfo.Methods = append(serviceInfo.Methods, methodInfo)
		}

		if len(serviceInfo.Methods) > 0 {
			data.Services = append(data.Services, serviceInfo)
		}
	}

	return data
}

// GenerateCode generates the code from templates.
func (g *Generator) GenerateCode(data *ServiceData) (string, error) {
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
		return g.extractPackageFromGoPackage(goPackage)
	}

	// Fall back to proto package name
	return g.extractPackageFromProtoPackage(file.GetPackage())
}

// extractPackageFromGoPackage extracts the package name from go_package option
func (g *Generator) extractPackageFromGoPackage(goPackage string) string {
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

// extractPackageFromProtoPackage extracts the package name from proto package
func (g *Generator) extractPackageFromProtoPackage(protoPackage string) string {
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
