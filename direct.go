package direct

import (
	"context"
	"net/http"
	"strings"
)

// directContextKey is the context key type for storing parameters in
// context.Context.
type directContextKey string

// Middleware is a function designed to run some code before and/or after
// another Handler.
type Middleware func(http.Handler) http.Handler

// Router routes HTTP requests.
type Router struct {
	routes []*route
	mw     []Middleware
	// NotFound is the http.Handler to call when no routes match. By default
	// uses http.NotFoundHandler().
	NotFound http.Handler
}

// NewRouter makes a new Router. Middleware is optional and will be executed by
// requests in the order they are provided.
func NewRouter(mw ...Middleware) *Router {
	return &Router{
		mw:       mw,
		NotFound: http.NotFoundHandler(),
	}
}

func pathSegments(pattern string) []string {
	return strings.Split(strings.Trim(pattern, "/"), "/")
}

// Handle adds a handler with the specified method, pattern and optional
// middleware. Method can be any HTTP method string or "*" to match all
// methods. Pattern can contain path segments such as: /item/:id which is
// accessible via the Param function. If pattern ends with trailing /, it acts
// as a prefix. Middleware is optional and will be executed by requests in the
// order they are provided.
func (r *Router) Handle(method, pattern string, handler http.Handler, mw ...Middleware) {

	// First, adapt handler specific middleware around this handler.
	handler = adapt(handler, mw...)

	// Then, adapt the application's general middleware to the handler chain.
	handler = adapt(handler, r.mw...)

	route := newRoute(method, pattern, handler)
	r.routes = append(r.routes, route)
}

// HandleFunc is the http.HandlerFunc alternative to http.Handle.
func (r *Router) HandleFunc(method, pattern string, fn http.HandlerFunc, mw ...Middleware) {
	r.Handle(method, pattern, fn, mw...)
}

// ServeHTTP routes the incoming http.Request based on method and path
// extracting path parameters as it goes.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	method := strings.ToLower(req.Method)
	for _, route := range r.routes {
		if route.method != method && route.method != "*" {
			continue
		}
		if ctx, ok := route.match(req.Context(), r, req.URL.Path); ok {
			route.handler.ServeHTTP(w, req.WithContext(ctx))
			return
		}
	}
	r.NotFound.ServeHTTP(w, req)
}

// Param gets the path parameter from the specified Context. Returns an empty
// string if the parameter was not found.
func Param(ctx context.Context, param string) string {
	vStr, ok := ctx.Value(directContextKey(param)).(string)
	if !ok {
		return ""
	}
	return vStr
}

// adapt creates a new Handler by wrapping middleware around a final handler.
// Middleware will be executed by requests in the order they are provided.
func adapt(h http.Handler, mw ...Middleware) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

type route struct {
	method  string
	segs    []string
	handler http.Handler
	prefix  bool
}

func newRoute(method, pattern string, handler http.Handler) *route {
	return &route{
		method:  strings.ToLower(method),
		segs:    pathSegments(pattern),
		handler: handler,
		prefix:  strings.HasSuffix(pattern, "/") || strings.HasSuffix(pattern, "..."),
	}
}

func (r *route) match(ctx context.Context, router *Router, path string) (context.Context, bool) {
	segs := pathSegments(path)
	if len(segs) > len(r.segs) && !r.prefix {
		return nil, false
	}
	for i, seg := range r.segs {
		if i > len(segs)-1 {
			return nil, false
		}
		isParam := false
		if strings.HasPrefix(seg, ":") {
			isParam = true
			seg = strings.TrimPrefix(seg, ":")
		}
		if !isParam { // verbatim check
			if strings.HasSuffix(seg, "...") {
				if strings.HasPrefix(segs[i], seg[:len(seg)-3]) {
					return ctx, true
				}
			}
			if seg != segs[i] {
				return nil, false
			}
		}
		if isParam {
			ctx = context.WithValue(ctx, directContextKey(seg), segs[i])
		}
	}
	return ctx, true
}
