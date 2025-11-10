# A multiplexer for Go

##### Supports

* Variables in the path
* The stdlib http.Handler interface (Also webassembly!)
* stdlib http.ResponseWriter/http.Request functions
* Route namespaces
* A generic `Multiplexer` interface compatible with both `*Mux` and `*Route`
* Various middlewares, including a custom [SCS session](github.com/alexedwards/scs/v2) middleware.

**For examples; please do not hesitate to look at the router_tests.go file.**
