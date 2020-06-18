package direct

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("%s:%d: "+msg+"\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("%s:%d: unexpected error: %s\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}

func TestRouting(t *testing.T) {
	var tests = []struct {
		RouteMethod  string
		RoutePattern string

		Method string
		Path   string
		Match  bool
		Params map[string]string
	}{
		// simple path matching
		{
			"GET", "/one",
			"GET", "/one", true, nil,
		},
		{
			"GET", "/two",
			"GET", "/two", true, nil,
		},
		{
			"GET", "/three",
			"GET", "/three", true, nil,
		},
		// methods
		{
			"get", "/methodcase",
			"GET", "/methodcase", true, nil,
		},
		{
			"Get", "/methodcase",
			"get", "/methodcase", true, nil,
		},
		{
			"GET", "/methodcase",
			"get", "/methodcase", true, nil,
		},
		{
			"GET", "/method1",
			"POST", "/method1", false, nil,
		},
		{
			"DELETE", "/method2",
			"GET", "/method2", false, nil,
		},
		{
			"GET", "/method3",
			"PUT", "/method3", false, nil,
		},
		// all methods
		{
			"*", "/all-methods",
			"GET", "/all-methods", true, nil,
		},
		{
			"*", "/all-methods",
			"POST", "/all-methods", true, nil,
		},
		{
			"*", "/all-methods",
			"PUT", "/all-methods", true, nil,
		},
		// nested
		{
			"GET", "/parent/child/one",
			"GET", "/parent/child/one", true, nil,
		},
		{
			"GET", "/parent/child/two",
			"GET", "/parent/child/two", true, nil,
		},
		{
			"GET", "/parent/child/three",
			"GET", "/parent/child/three", true, nil,
		},
		// slashes
		{
			"GET", "slashes/one",
			"GET", "/slashes/one", true, nil,
		},
		{
			"GET", "/slashes/two",
			"GET", "slashes/two", true, nil,
		},
		{
			"GET", "slashes/three/",
			"GET", "/slashes/three", true, nil,
		},
		{
			"GET", "/slashes/four",
			"GET", "slashes/four/", true, nil,
		},
		// prefix
		{
			"GET", "/prefix/",
			"GET", "/prefix/anything/else", true, nil,
		},
		{
			"GET", "/not-prefix",
			"GET", "/not-prefix/anything/else", false, nil,
		},
		{
			"GET", "/prefixdots...",
			"GET", "/prefixdots/anything/else", true, nil,
		},
		{
			"GET", "/prefixdots...",
			"GET", "/prefixdots", true, nil,
		},
		// path params
		{
			"GET", "/path-param/:id",
			"GET", "/path-param/123", true, map[string]string{"id": "123"},
		},
		{
			"GET", "/path-params/:era/:group/:member",
			"GET", "/path-params/60s/beatles/lennon", true, map[string]string{
				"era":    "60s",
				"group":  "beatles",
				"member": "lennon",
			},
		},
		{
			"GET", "/path-params-prefix/:era/:group/:member/",
			"GET", "/path-params-prefix/60s/beatles/lennon/yoko", true, map[string]string{
				"era":    "60s",
				"group":  "beatles",
				"member": "lennon",
			},
		},
		// misc no matches
		{
			"GET", "/not/enough",
			"GET", "/not/enough/items", false, nil,
		},
		{
			"GET", "/not/enough/items",
			"GET", "/not/enough", false, nil,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			r := NewRouter()
			match := false
			var ctx context.Context
			r.Handle(test.RouteMethod, test.RoutePattern, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				match = true
				ctx = r.Context()
			}))
			req, err := http.NewRequest(test.Method, test.Path, nil)
			ok(t, err)
			r.ServeHTTP(httptest.NewRecorder(), req)
			assert(t, match == test.Match, fmt.Sprintf("expected match %v but was %v: %s %s", test.Match, match, test.Method, test.Path))
			if len(test.Params) > 0 {
				for expK, expV := range test.Params {
					// check using helper
					actV := Param(ctx, expK)
					assert(t, actV == expV, fmt.Sprintf("Param: context value %s expected \"%s\" but was \"%s\"", expK, expV, actV))
				}
			}
		})
	}
}

func TestMultipleRoutesDifferentMethods(t *testing.T) {
	r := NewRouter()
	var match string
	r.Handle(http.MethodGet, "/route", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		match = "GET /route"
	}))
	r.Handle(http.MethodDelete, "/route", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		match = "DELETE /route"
	}))
	r.Handle(http.MethodPost, "/route", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		match = "POST /route"
	}))
	r.Handle(http.MethodPut, "/route", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		match = "PUT /route"
	}))

	req, err := http.NewRequest(http.MethodGet, "/route", nil)
	ok(t, err)
	r.ServeHTTP(httptest.NewRecorder(), req)
	assert(t, match == "GET /route", fmt.Sprintf("unexpected: %s", match))

	req, err = http.NewRequest(http.MethodDelete, "/route", nil)
	ok(t, err)
	r.ServeHTTP(httptest.NewRecorder(), req)
	assert(t, match == "DELETE /route", fmt.Sprintf("unexpected: %s", match))

	req, err = http.NewRequest(http.MethodPost, "/route", nil)
	ok(t, err)
	r.ServeHTTP(httptest.NewRecorder(), req)
	assert(t, match == "POST /route", fmt.Sprintf("unexpected: %s", match))

	req, err = http.NewRequest(http.MethodPut, "/route", nil)
	ok(t, err)
	r.ServeHTTP(httptest.NewRecorder(), req)
	assert(t, match == "PUT /route", fmt.Sprintf("unexpected: %s", match))
}

type testLogger struct{ history []string }

func (l *testLogger) log(s string) { l.history = append(l.history, s) }

func notify(l *testLogger, prefix string) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l.log(fmt.Sprintf("%s: before", prefix))
			defer l.log(fmt.Sprintf("%s: after", prefix))
			h.ServeHTTP(w, r)
		})
	}
}

func TestMiddlewareExecutedInOrderAsProvided(t *testing.T) {
	logger := &testLogger{}
	teardown := func() {
		logger.history = nil
	}

	var tests = []struct {
		routerMw  []Middleware
		handlerMw []Middleware

		logger     *testLogger
		expLogHist []string
	}{
		{
			routerMw:   []Middleware{notify(logger, "routerMw")},
			handlerMw:  nil,
			logger:     logger,
			expLogHist: []string{"routerMw: before", "routerMw: after"},
		},
		{
			routerMw:   nil,
			handlerMw:  []Middleware{notify(logger, "handlerMw")},
			logger:     logger,
			expLogHist: []string{"handlerMw: before", "handlerMw: after"},
		},
		{
			routerMw:  []Middleware{notify(logger, "routerMw1"), notify(logger, "routerMw2")},
			handlerMw: []Middleware{notify(logger, "handlerMw1"), notify(logger, "handlerMw2")},
			logger:    logger,
			expLogHist: []string{
				"routerMw1: before",
				"routerMw2: before",
				"handlerMw1: before",
				"handlerMw2: before",
				"handlerMw2: after",
				"handlerMw1: after",
				"routerMw2: after",
				"routerMw1: after",
			},
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			defer teardown()
			r := NewRouter(tc.routerMw...)
			r.HandleFunc(http.MethodGet, "/route", func(w http.ResponseWriter, r *http.Request) {}, tc.handlerMw...)
			req, err := http.NewRequest(http.MethodGet, "/route", nil)
			ok(t, err)
			r.ServeHTTP(httptest.NewRecorder(), req)
			equals(t, tc.logger.history, tc.expLogHist)
		})
	}
}
