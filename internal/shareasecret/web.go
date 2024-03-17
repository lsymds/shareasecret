package shareasecret

import "net/http"

type httpServer struct {
	application *Application
}

// ServeHTTP implements the http.Handler interface for the Application struct, allowing it to be used as the core handler
// of a HTTP server.
func (a *Application) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (h *httpServer) handleGetIndex(w http.ResponseWriter, r *http.Request) {}

func (h *httpServer) handleCreateSecret(w http.ResponseWriter, r *http.Request) {}

func (h *httpServer) handleDeleteSecret(w http.ResponseWriter, r *http.Request) {}

func (h *httpServer) handleGetSecret(w http.ResponseWriter, r *http.Request) {}

func (h *httpServer) handleReadSecret(w http.ResponseWriter, r *http.Request) {}
