package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type (
	server struct {
		router   *mux.Router
		httpSrv  *http.Server
		logger   *log.Logger
		versions map[string][]route
		c        chan os.Signal
	}

	route struct {
		url        string
		methods    []string
		controller func(http.ResponseWriter, *http.Request)
	}

	// dbRes is just a container the basic resources needed to do
	// basic database work.
	dbRes struct {
		ctx context.Context
		db  *sql.DB
	}

	Key int
)

var (
	// this indicates up or down status of server
	// > 1 means server up and 0 means down.
	SERVER_HEALTH int32
	globalRes     *dbRes //

	nextRequestID = func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
)

const (

	// https://stackoverflow.com/questions/39946583/how-to-pass-context-in-golang-request-to-middleware
	// https://blog.golang.org/context
	// https://medium.com/@cep21/how-to-correctly-use-context-context-in-go-1-7-8f2c0fafdf39#.7dcv2847z
	// https://blog.golang.org/context#TOC_3.1.

	requestIDKey Key = 0
)

func (srv *server) newVersion(versionPrefix string, urls []route) {
	srv.versions[versionPrefix] = urls
}

func (srv *server) registerRoutes() {
	srv.router.HandleFunc("/health", checkServerStatus).Methods("GET")

	for version, routes := range srv.versions {
		for _, r := range routes {
			srv.router.HandleFunc(version+r.url, r.controller).Methods(r.methods...)
		}
	}

	srv.registerMiddlewares()
	//srv.router.PathPrefix("/api/v0/")
}

func (srv *server) setupLogger() {
	srv.logger = log.New(os.Stdout, "Default: ", log.LstdFlags)
}

func (srv *server) registerMiddlewares() {
	srv.router.Use(
		enableTracing(nextRequestID),
		logIncomingReqDetails(srv.logger),
	)
}

func (srv *server) startServer(port string) {
	// do some initialization before starting server.
	srv.setupLogger()
	srv.registerRoutes()

	srv.httpSrv = &http.Server{
		Handler:      srv.router,
		Addr:         "127.0.0.1:" + port,
		ErrorLog:     srv.logger,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	// run server is new goroutine to prevent blocking.
	go func() {
		srv.logger.Printf("HTTP server started on http://localhost:%s/", os.Getenv("PORT"))
		if err := srv.httpSrv.ListenAndServe(); err != nil {
			srv.logger.Println(err)
		}
	}()

	atomic.StoreInt32(&SERVER_HEALTH, 1)
}

func (srv *server) shutdown() {
	signal.Notify(srv.c, os.Interrupt)
	// block until signal is recieved.
	<-srv.c

	srv.logger.Println("Server gracefully shutting down...")
	atomic.StoreInt32(&SERVER_HEALTH, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	srv.httpSrv.SetKeepAlivesEnabled(false)
	if err := srv.httpSrv.Shutdown(ctx); err != nil {
		srv.logger.SetPrefix("Error: ")
		srv.logger.Fatalf("Could not gracefully shutdown server: %v\n", err)
	}

	os.Exit(0)
}

// Api Version 0 Routes
func V0RoutesAndCtrls() []route {
	return []route{
		{
			url:        "/register",
			methods:    []string{"POST"},
			controller: V0_RegisterUser,
		},
		{
			url:        "/login",
			methods:    []string{"POST"},
			controller: V0_LoginUser,
		},
		{
			url:        "/{user}/friends",
			methods:    []string{"GET"},
			controller: V0_GetFriends,
		},
	}
}

// Controller Functions
func V0_RegisterUser(w http.ResponseWriter, r *http.Request) {
	Json(w, http.StatusNotImplemented, struct {
		Message string `json:"msg"`
	}{
		Message: "Not yet implemented",
	})
}

func V0_LoginUser(w http.ResponseWriter, r *http.Request) {
	Error(w, http.StatusNotImplemented, errors.New("not logged in\n"))
}

func V0_GetFriends(w http.ResponseWriter, r *http.Request) {
	Json(w, http.StatusNotImplemented, struct {
		Message string `json:"msg"`
	}{
		Message: fmt.Sprintf("You will have to waita lil bit %s!",
			mux.Vars(r)["user"]),
	})
}

func checkServerStatus(w http.ResponseWriter, r *http.Request) {
	if atomic.LoadInt32(&SERVER_HEALTH) == 1 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
}

//response functions. This functions serialize data to json and send to
// an output stream.
func Json(w http.ResponseWriter, statusCode int, data interface{}) {
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Fprintf(w, "%s", err.Error())
	}
}

func Error(w http.ResponseWriter, statusCode int, err error) {
	Json(w, statusCode, struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	})
}

// ------------ Middlewares -----------------------
func enableTracing(nextId func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestId := r.Header.Get("X-Request-Id")
			if requestId == "" {
				requestId = nextId()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, requestId)
			w.Header().Set("X-Request-Id", requestId)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func logIncomingReqDetails(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestId, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestId = "unknown"
				}
				logger.SetPrefix(r.Proto + " ")
				logger.Println(requestId, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// NewServer returns a new server for use.
func NewServer() *server {
	return &server{
		router:   mux.NewRouter().StrictSlash(true),
		versions: make(map[string][]route),
		c:        make(chan os.Signal, 1),
	}
}

// ----------- MAIN AND INIT FUNCTIONS -----------------------------
func init() {
	godotenv.Load()
}

func main() {
	// init new server and start serving.
	srv := NewServer()
	srv.newVersion("/api/v0", V0RoutesAndCtrls()) //register version of routes to serve
	srv.startServer(os.Getenv("PORT"))
	defer srv.shutdown() // wait to recieve interrupt signals and shutdown server.
}

// -------------------------------------------------------------------
// connect opens connection to the database
func connect() *sql.DB {
	return nil
}

// initialize a global dbRes
func Init() {
	globalRes = &dbRes{
		db:  connect(),
		ctx: context.Background(),
	}
}
