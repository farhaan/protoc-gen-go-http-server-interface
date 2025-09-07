// Package internal provides helper utilities for testing the protoc-gen-go-http-server-interface plugin.
package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// ProtoTemplateData holds data for proto template rendering
type ProtoTemplateData struct {
	Package     string
	GoPackage   string
	ServiceName string
}

// GenerateProtoFromTemplate generates a proto file from a template
func GenerateProtoFromTemplate(templateName, outputPath string, data ProtoTemplateData) error {
	// Find the correct path to the template file
	templatePath := filepath.Join("testdata", "templates", templateName+".proto.tmpl")

	// If not found, try parent directory (for subdirectory tests)
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		templatePath = filepath.Join("..", "testdata", "templates", templateName+".proto.tmpl")
	}

	tmplContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %v", templatePath, err)
	}

	// Parse and execute the template
	tmpl, err := template.New(templateName).Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	// Create the output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	// Execute the template
	err = tmpl.Execute(outputFile, data)
	if err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	return nil
}

// GenerateBasicServiceProto generates a basic service proto file
func GenerateBasicServiceProto(outputPath, packageName, serviceName string) error {
	data := ProtoTemplateData{
		Package:     packageName,
		GoPackage:   fmt.Sprintf("github.com/example/%s;%s", packageName, packageName),
		ServiceName: serviceName,
	}
	return GenerateProtoFromTemplate("basic_service", outputPath, data)
}

// GenerateAdvancedServiceProto generates an advanced service proto with multiple HTTP methods
func GenerateAdvancedServiceProto(outputPath, packageName, serviceName string) error {
	data := ProtoTemplateData{
		Package:     packageName,
		GoPackage:   fmt.Sprintf("github.com/test/%s;%s", packageName, packageName),
		ServiceName: serviceName,
	}
	return GenerateProtoFromTemplate("advanced_service", outputPath, data)
}

// GenerateMultiServiceProto generates a multi-service proto file
func GenerateMultiServiceProto(outputPath, packageName string) error {
	data := ProtoTemplateData{
		Package:   packageName,
		GoPackage: fmt.Sprintf("github.com/example/%s;%s", packageName, packageName),
	}
	return GenerateProtoFromTemplate("multi_service", outputPath, data)
}

// GenerateCompatibilityServiceProto generates a compatibility service proto file
func GenerateCompatibilityServiceProto(outputPath, packageName, serviceName string) error {
	data := ProtoTemplateData{
		Package:     packageName,
		GoPackage:   fmt.Sprintf("github.com/example/%s;%s", packageName, packageName),
		ServiceName: serviceName,
	}
	return GenerateProtoFromTemplate("compatibility_service", outputPath, data)
}
