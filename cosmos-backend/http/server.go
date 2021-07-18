package http

import (
	"context"
	"cosmos"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigDefault

// Server represents a HTTP server.
type Server struct {
	listener net.Listener
	server   *http.Server
	router   *mux.Router

	Addr string

	*cosmos.App
}

// spaHandler implements the http.Handler interface, so we can use it
// to respond to HTTP requests. The path to the static directory and
// path to the index file within that static directory are used to
// serve the SPA in the given static directory.
type spaHandler struct {
	staticPath string
	indexPath  string
}

// ServeHTTP inspects the URL path to locate a file within the static dir
// on the SPA handler. If a file is found, it will be served. If not, the
// file located at the index path on the SPA handler will be served. This
// is suitable behavior for serving an SPA (single page application).
func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get the absolute path to prevent directory traversal.
	path, err := filepath.Abs(r.URL.Path)
	if err != nil {
		// If we failed to get the absolute path respond with a 400 bad request and stop.
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Prepend the path with the path to the static directory.
	path = filepath.Join(h.staticPath, path)

	// Check whether a file exists at the given path.
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		// File does not exist, serve index.html.
		http.ServeFile(w, r, filepath.Join(h.staticPath, h.indexPath))
		return
	} else if err != nil {
		// If we got an error (that wasn't that the file doesn't exist) stating the
		// file, return a 500 internal server error and stop.
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Otherwise, use http.FileServer to serve the static dir.
	http.FileServer(http.Dir(h.staticPath)).ServeHTTP(w, r)
}

// NewServer returns a new instance of Server.
func NewServer(addr string) *Server {
	s := &Server{
		server: &http.Server{},
		router: mux.NewRouter(),
		Addr:   addr,
	}

	// Delegate HTTP handling to the Gorilla router.
	// Allow CORS (See https://www.thepolyglotdeveloper.com/2017/10/handling-cors-golang-web-application/).
	s.server.Handler = handlers.CORS(
		handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}),
		handlers.AllowedOrigins([]string{"*"}),
	)(s.router)

	// Register routes.
	r := s.router.PathPrefix("/api/v1").Subrouter()
	r.Use(recoveryMiddleware)
	s.registerConnectorRoutes(r)
	s.registerEndpointRoutes(r)
	s.registerSyncRoutes(r)
	s.registerRunRoutes(r)
	s.registerArtifactRoutes(r)

	// Serve SPA (Single Page Application).
	// See https://github.com/gorilla/mux#serving-single-page-applications
	spa := spaHandler{staticPath: "dist", indexPath: "index.html"}
	s.router.PathPrefix("/").Handler(spa)

	return s
}

// recoverMiddleware recovers from panics in HTTP handlers.
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[http] error: %s %s: %s", r.Method, r.URL.Path, err)
				debug.PrintStack()
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// Open begins listening on the bind address.
func (s *Server) Open() (err error) {
	if s.listener, err = net.Listen("tcp", s.Addr); err != nil {
		return err
	}

	// Begin serving requests on a listener. We use Serve() instead of
	// ListenAndServe() because it allows us to check for listener errors
	// (such as trying to use an already open port) synchronously.
	go s.server.Serve(s.listener)

	return nil
}

// Close gracefully shuts down the HTTP server.
func (s *Server) Close() error {
	if s.listener != nil {
		// Allow 30 seconds for the server to shutdown cleanly.
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

// LogError logs an internal error.
func (s *Server) LogError(r *http.Request, err error) {
	log.Printf("[http] error: %s %s: %s", r.Method, r.URL.Path, err)
}

// ReplyWithSanitizedError sends a failure response making sure to hide sensitive internal errors
// from the end user. All errors returned by the application code are, by default, internal errors.
// If the application code wants to send a non-internal-error to the end-user, it must explicitly
// return a cosmos.Error with the appropriate code.
func (s *Server) ReplyWithSanitizedError(w http.ResponseWriter, r *http.Request, err error) {
	code, message := cosmos.ErrorCode(err), cosmos.ErrorMessage(err)

	// Log the error for application developers to examine.
	if code == cosmos.EINTERNAL {
		s.LogError(r, err)
	}

	// Send sanitized errors in the response. For internal errors, this means "Internal error".
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(ErrorStatusCode(code))
	if err := json.NewEncoder(w).Encode(&ErrorResponse{Error: message}); err != nil {
		s.LogError(r, err)
	}
}

// ErrorResponse represents the sanitized error sent in a response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ErrorStatusCode returns the HTTP status code that matches the application-specific error code.
func ErrorStatusCode(code string) int {
	codes := map[string]int{
		cosmos.EINVALID:        http.StatusBadRequest,
		cosmos.EINTERNAL:       http.StatusInternalServerError,
		cosmos.ECONFLICT:       http.StatusConflict,
		cosmos.ENOTFOUND:       http.StatusNotFound,
		cosmos.ENOTIMPLEMENTED: http.StatusNotImplemented,
	}

	if statusCode, ok := codes[code]; ok {
		return statusCode
	}

	return http.StatusInternalServerError
}
