// Package internal provides helper utilities for testing the protoc-gen-go-http-server-interface plugin.
package internal

import (
	"os"
	"os/exec"
)

// FindPluginBinary locates the protoc plugin binary
func FindPluginBinary() (string, error) {
	// Check in PATH (installed via go install)
	return exec.LookPath("protoc-gen-go-http-server-interface")
}

// HasBuf checks if buf is available in the system
func HasBuf() bool {
	_, err := exec.LookPath("buf")
	return err == nil
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
