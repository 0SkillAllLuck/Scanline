package router

import "strings"

// Route represents a registered route with its pattern and handler.
type Route struct {
	pattern string
	nParams int
}

// NewRoute creates a new route, validates the handler, and registers it.
// The handler must be a function whose first parameter is *appctx.AppContext,
// followed by string parameters matching the number of path params in the pattern,
// and returning *Response.
func NewRoute(pattern string, handler any) *Route {
	r := &Route{
		pattern: pattern,
	}

	// Count params and extract segments
	parts := strings.Split(pattern, "/")
	for _, p := range parts {
		if strings.HasPrefix(p, ":") {
			r.nParams++
		}
	}

	Register(pattern, handler)
	return r
}

// Path builds a URL path string from the route pattern and provided arguments.
func (r *Route) Path(args ...string) string {
	parts := strings.Split(r.pattern, "/")
	result := make([]string, 0, len(parts))
	argIdx := 0
	for _, p := range parts {
		if strings.HasPrefix(p, ":") {
			if argIdx < len(args) {
				result = append(result, args[argIdx])
				argIdx++
			}
		} else {
			result = append(result, p)
		}
	}
	return strings.Join(result, "/")
}

// Pattern returns the route's pattern string.
func (r *Route) Pattern() string {
	return r.pattern
}
