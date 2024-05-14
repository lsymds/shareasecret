package shareasecret

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// deletionReasonExpired is a deletion reason used when secrets have exceeded their TTL (time to live)
const deletionReasonExpired = "expired"

// deletionReasonUserDeleted is a deletion reason used when a user actions the deletion themselves
const deletionReasonUserDeleted = "user_deleted"

// deletionReasonMaximumViewCountHit is a deletion reason used when the maximum number of views for a secret has been
// hit or exceeded
const deletionReasonMaximumViewCountHit = "maximum_view_count_hit"

// Configuration contains all of the possible configuration options for the application.
type Configuration struct {
	Database struct {
		Path string
	}
	Server struct {
		BaseUrl       string
		ListeningAddr string
	}
	SecretCreationRestrictions struct {
		IPAddresses []string
	}
}

// PopulateFromEnv populates all of the configuration values from environment variables, returning errors if this
// cannot be achieved.
func (c *Configuration) PopulateFromEnv() error {
	err := godotenv.Load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("loading env file: %w", err)
	}

	c.Database.Path = os.Getenv("SHAREASECRET_DB_PATH")
	if c.Database.Path == "" {
		return fmt.Errorf("SHAREASECRET_DB_PATH not set")
	}

	c.Server.BaseUrl = os.Getenv("SHAREASECRET_BASE_URL")
	if c.Server.BaseUrl == "" {
		return fmt.Errorf("SHAREASECRET_BASE_URL not set")
	}

	c.Server.ListeningAddr = os.Getenv("SHAREASECRET_LISTENING_ADDR")
	if c.Server.ListeningAddr == "" {
		c.Server.ListeningAddr = "127.0.0.1:8994"
	}

	if cr := strings.TrimSpace(os.Getenv("SHAREASECRET_SECRET_CREATION_IP_RESTRICTIONS")); cr != "" {
		c.SecretCreationRestrictions.IPAddresses = strings.Split(cr, ",")
	}

	return nil
}

// Application is a wrapper/container for the "ShareASecret" project. All jobs and entry points hang off of this
// struct.
type Application struct {
	db        *database
	config    *Configuration
	router    *http.ServeMux
	baseURL   string
	webAssets fs.FS
}

// NewApplication initializes the Application struct which provides access to all available components of the project.
func NewApplication(config *Configuration, webAssets fs.FS) (*Application, error) {
	db, err := newDatabase("file:" + config.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("new db: %w", err)
	}

	application := &Application{
		db:        db,
		config:    config,
		router:    http.NewServeMux(),
		baseURL:   config.Server.BaseUrl,
		webAssets: webAssets,
	}
	application.mapRoutes()

	return application, nil
}
