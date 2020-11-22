## DEMO FOR A BIGGER PROJECT

I did this to help me visualize the ideas i had for a bigger project. I had this structure of a server is mind but had to test it out to see how it will work in real live before trying to do anything.

The code is structured in a way to help me start new servers with little effort. If I decide to use microservice architecture, i would just turn the whole thing into a package that i can use to spring up new http servers with very little effort.

The most important types in this app are;
```go
type server struct {
	router   *mux.Router
	httpSrv  *http.Server
	logger   *log.Logger
	versions map[string][]route
	c        chan os.Signal
}
```
and the
```go
type route struct {
	url        string
	methods     []string
	controller func(http.ResponseWriter, *http.Request)
}
```
I'm thinking of making the `controller` accept context so its type declaration will change to something like `func(context.Context) func(http.ResponseWriter, *http.Request)`, but i'm still struggling with the whole idea of the context package soo i'll hold off the idea for a bit. Ohhh, `*http.Request` already has `context.Context` type on it so making the `controller` accept additional `context.Context` will be a stupid thing to do.


Might fuck around and make this whole into a package i can use to spawn http servers fast. But i won't like
to use the same logic in multiple servers since i'll have to learn other ways of doing this stuff.
