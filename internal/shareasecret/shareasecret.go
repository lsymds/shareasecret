package shareasecret

import (
	"fmt"
	"io/fs"
	"net/http"
)

// deletionReasonExpired is a deletion reason used when secrets have exceeded their TTL (time to live)
const deletionReasonExpired = "expired"

// deletionReasonUserDeleted is a deletion reason used when a user actions the deletion themselves
const deletionReasonUserDeleted = "user_deleted"

// deletionReasonMaximumViewCountHit is a deletion reason used when the maximum number of views for a secret has been
// hit or exceeded
const deletionReasonMaximumViewCountHit = "maximum_view_count_hit"

// Application is a wrapper/container for the "ShareASecret" project. All jobs and entry points hang off of this
// struct.
type Application struct {
	db        *database
	router    *http.ServeMux
	baseURL   string
	webAssets fs.FS
}

// NewApplication initializes the Application struct which provides access to all available components of the project.
func NewApplication(connectionString string, baseURL string, webAssets fs.FS) (*Application, error) {
	db, err := newDatabase(connectionString)
	if err != nil {
		return nil, fmt.Errorf("new db: %w", err)
	}

	application := &Application{
		db:        db,
		router:    http.NewServeMux(),
		baseURL:   baseURL,
		webAssets: webAssets,
	}
	application.mapRoutes()

	return application, nil
}
