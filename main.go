// Command protoc-gen-httpinterface is a plugin for the Google protocol buffer compiler to generate
// HTTP interface code. It is linked into protoc at runtime.
//
// Usage:
//
//	protoc --httpinterface_out=. path/to/file.proto
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"google.golang.org/protobuf/proto"

	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface"
)

func main() {
	// Flags can be used for plugin options
	var flags flag.FlagSet
	flags.Parse(os.Args[1:])

	// Read input from stdin (protoc pipes input here)
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		logFatal(err, "unable to read input")
	}

	// Parse the input as a protoc CodeGeneratorRequest
	var request plugin.CodeGeneratorRequest
	if err := proto.Unmarshal(data, &request); err != nil {
		logFatal(err, "unable to parse input")
	}

	// Create a new httpinterface generator
	g := httpinterface.New()

	// Generate the code
	response := g.Generate(&request)

	// Marshal the response
	output, err := proto.Marshal(response)
	if err != nil {
		logFatal(err, "failed to marshal output")
	}

	// Write the output to stdout (protoc reads it from here)
	if _, err := os.Stdout.Write(output); err != nil {
		logFatal(err, "failed to write output")
	}
}

func logFatal(err error, msg string) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	os.Exit(1)
}
