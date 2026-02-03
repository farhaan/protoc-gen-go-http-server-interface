// Package stdlib demonstrates using generated code with Go 1.22+ http.ServeMux.
//
// The stdlib ServeMux supports "METHOD /path" patterns natively, making it
// the simplest option with zero dependencies.
package stdlib

import "net/http"

// Routes wraps http.ServeMux to implement the pb.Routes interface.
// This is a minimal wrapper - just 1 method.
type Routes struct {
	Mux *http.ServeMux
}

// HandleFunc registers a handler for the given method and pattern.
// Uses Go 1.22+ "METHOD /path" pattern syntax.
func (r Routes) HandleFunc(method, pattern string, handler http.HandlerFunc) {
	r.Mux.HandleFunc(method+" "+pattern, handler)
}

// New creates a new Routes with the given mux.
// If mux is nil, a new http.ServeMux is created.
func New(mux *http.ServeMux) Routes {
	if mux == nil {
		mux = http.NewServeMux()
	}
	return Routes{Mux: mux}
}
