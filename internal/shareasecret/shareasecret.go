package shareasecret

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
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
		IPAddresses struct {
			FixedIPs []net.IP
			CIDRs    []net.IPNet
		}
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
		for _, v := range strings.Split(cr, ",") {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}

			// a slash in the IP address consistutes a CIDR i.e. fe80::/8 or 192.168.1.0/24
			if strings.Contains(v, "/") {
				_, nw, err := net.ParseCIDR(v)
				if err != nil {
					return fmt.Errorf("invalid CIDR (%v) in SHAREASECRET_SECRET_CREATION_IP_RESTRICTIONS: %w", v, err)
				}

				c.SecretCreationRestrictions.IPAddresses.CIDRs = append(c.SecretCreationRestrictions.IPAddresses.CIDRs, *nw)
			} else {
				if ip := net.ParseIP(v); ip == nil {
					return fmt.Errorf("invalid ip in SHAREASECRET_SECRET_CREATION_IP_RESTRICTIONS: %v", v)
				} else {
					c.SecretCreationRestrictions.IPAddresses.FixedIPs = append(c.SecretCreationRestrictions.IPAddresses.FixedIPs, ip)
				}
			}
		}
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
