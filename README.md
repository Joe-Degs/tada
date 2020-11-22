## DEMO FOR A BIGGER PROJECT

I did this to help me visualize the ideas i had for a bigger project. I had this structure of an http server in mind but had to test it out to see how it will work out before starting the project.

The code is structured in a way to help me start new servers with little effort. If I decide to use microservice architecture, i would just turn the whole thing into a package that i can use to spring up new http servers with very little effort.

The most important types in this app are;
```go
type server struct {
	router   *mux.Router
	httpSrv  *http.Server
	logger   *log.Logger
	
	// most api's have versioning in the form of /api/v1/something. I call the '/api/v1' the 
	// versionPrefix because it will be prefixed on the url of a route type. I wanted to
	// have that in the server and the best way I could achieve that was using a map to
	// keep track of a versionPrefix and the slice of routes that version
	// is released with. So I can initialize the routes by adding the versionPrefix to every
	// every url in the route associated with it.
	versions map[string][]route

	c        chan os.Signal // used to recieve interrupt signals to gracefully shutdown server.
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

Still thinking about how the database interfacing is going to be like. Arggh thats the hardest part for me right now since i have very little experience with databases. I haven't fully decided on whether to use
an orm or go with the `database/sql` libary.

Might fuck around and make this whole thing into a package I can use to spawn http servers fast. But i won't like to use the same logic in multiple projects since i'll have to learn other ways of doing this kinda stuff.
