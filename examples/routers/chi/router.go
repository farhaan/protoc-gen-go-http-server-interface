// Package chi demonstrates using generated code with go-chi/chi router.
//
// Chi is a lightweight, idiomatic router for Go. This example shows
// how to create a minimal wrapper to use chi with generated code.
//
// Note: Chi uses chi.URLParam(r, "param") for path parameters, not r.PathValue().
// Your handler implementations need to use chi.URLParam accordingly.
package chi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes wraps chi.Router to implement the pb.Routes interface.
type Routes struct {
	Router chi.Router
}

// HandleFunc registers a handler for the given method and pattern.
func (c Routes) HandleFunc(method, pattern string, handler http.HandlerFunc) {
	c.Router.MethodFunc(method, pattern, handler)
}

// New creates a new Routes with the given router.
// If router is nil, a new chi.Router is created.
func New(router chi.Router) Routes {
	if router == nil {
		router = chi.NewRouter()
	}
	return Routes{Router: router}
}
