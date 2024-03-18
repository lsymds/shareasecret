package shareasecret

import (
	"fmt"
	"net/http"
)

// mapRoutes maps all HTTP routes for the application.
func (a *Application) mapRoutes() {
	fs := http.FileServer(http.Dir("./static/"))
	a.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	a.router.HandleFunc("/", a.handleGetIndex).Methods("GET")
	a.router.HandleFunc("/secret", a.handleCreateSecret).Methods("POST")
	a.router.HandleFunc("/secret/{secretId}", a.handleGetSecret).Methods("GET")
	a.router.HandleFunc("/secret/{secretId}/view", a.handleReadSecret).Methods("GET")
	a.router.HandleFunc("/manage-secret/{managementId}", a.handleManageSecret).Methods("GET")
	a.router.HandleFunc("/manage-secret/{managementId}", a.handleDeleteSecret).Methods("DELETE")
}

func (a *Application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

func (a *Application) handleGetIndex(w http.ResponseWriter, r *http.Request) {
	pageIndex().Render(r.Context(), w)
}

func (a *Application) handleCreateSecret(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("creating a secret")
}

func (a *Application) handleGetSecret(w http.ResponseWriter, r *http.Request) {}

func (a *Application) handleReadSecret(w http.ResponseWriter, r *http.Request) {}

func (a *Application) handleManageSecret(w http.ResponseWriter, r *http.Request) {}

func (a *Application) handleDeleteSecret(w http.ResponseWriter, r *http.Request) {}
