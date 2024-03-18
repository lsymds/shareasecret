package shareasecret

import (
	"fmt"

	"github.com/gorilla/mux"
)

// Application is a wrapper/container for the "ShareASecret" project. All jobs and entry points hang off of this
// struct.
type Application struct {
	db     *database
	router *mux.Router
}

// NewApplication initializes the Application struct which provides access to all available components of the project.
func NewApplication(connectionString string) (*Application, error) {
	db, err := newDatabase(connectionString)
	if err != nil {
		return nil, fmt.Errorf("new db: %w", err)
	}

	application := &Application{
		db:     db,
		router: mux.NewRouter(),
	}
	application.mapRoutes()

	return application, nil
}