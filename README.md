# Direct

The router found in this package is an extension of [way](https://github.com/matryer/way) — a simple HTTP router in Go by [Mat Ryer](https://github.com/matryer). This package is practically the same, but additionally supports optional middleware; both general to the router and handler specific.

## Install

There's no need to add a dependency to Direct, just copy `direct.go` and `direct_test.go` into your project.

If you prefer, it is go gettable:

```
go get github.com/johanronkko/direct
```

## Usage

* Use `NewRouter` to make a new `Router` with optional middleware
* Call `Handle` and `HandleFunc` to add handlers
* Specify HTTP method, path pattern for each route and optional middleware
* Use `Param` function to get the path parameters from the context

```go
func notify(l *log.Logger) direct.Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l.Println(fmt.Sprintf("before"))
			defer l.Println(fmt.Sprintf("after"))
			h.ServeHTTP(w, r)
		})
	}
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	msg := direct.Param(r.Context(), "msg")
	fmt.Fprintf(w, "Pong: %s", msg)
}

func main() {
	l1 := log.New(os.Stdout, "l1: ", log.Ldate|log.Ltime|log.Lshortfile)
	l2 := log.New(os.Stdout, "l2: ", log.Ldate|log.Ltime|log.Lshortfile)
	r := direct.NewRouter(notify(l1))
	r.HandleFunc("GET", "/ping/:msg", handlePing, notify(l2))
	log.Fatalln(http.ListenAndServe(":8080", r))
}
```

* Prefix matching

To match any path that has a specific prefix, use the `...` prefix indicator:

```go
func main() {
	r := direct.NewRouter()
	r.HandleFunc("GET", "/images...", handleImages)
	log.Fatalln(http.ListenAndServe(":8080", r))
}
```

In the above example, the following paths will match:

* `/images`
* `/images/`
* `/images/one/two/three.jpg`

* Set `Router.NotFound` to handle 404 errors manually

```go
func main() {
	r := direct.NewRouter()
	r.NotFound = http.HandlerFunc(func(w http ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "This is not the page you are looking for")
	})
	log.Fatalln(http.ListenAndServe(":8080", r))
}
```

## Copyright notices
``` 
Copyright (c) 2016 Mat Ryer
```
```
Copyright (c) 2020 Johan Rönkkö
```