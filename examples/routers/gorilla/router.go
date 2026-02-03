// Package gorilla demonstrates using generated code with gorilla/mux router.
//
// Gorilla mux is a powerful URL router and dispatcher. This example shows
// how to create a minimal wrapper to use gorilla/mux with generated code.
//
// Note: Gorilla uses mux.Vars(r)["param"] for path parameters, not r.PathValue().
// Your handler implementations need to use mux.Vars accordingly.
package gorilla

import (
	"net/http"

	"github.com/gorilla/mux"
)

// Routes wraps mux.Router to implement the pb.Routes interface.
type Routes struct {
	Router *mux.Router
}

// HandleFunc registers a handler for the given method and pattern.
func (g Routes) HandleFunc(method, pattern string, handler http.HandlerFunc) {
	g.Router.HandleFunc(pattern, handler).Methods(method)
}

// New creates a new Routes with the given router.
// If router is nil, a new mux.Router is created.
func New(router *mux.Router) Routes {
	if router == nil {
		router = mux.NewRouter()
	}
	return Routes{Router: router}
}
