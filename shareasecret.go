package main

import (
	"embed"
	"io/fs"
	"net/http"
	"os"

	"github.com/lsymds/shareasecret/internal/shareasecret"
	"github.com/rs/zerolog/log"
)

//go:embed web/**
var embeddedWebAssets embed.FS

// version defines the version of the application, used for informational purposes when logging.
//
// It is overwritten by utilising build tags at build time.
var version string = "0.0.1"

func main() {
	log.Info().Str("version", version).Msg("starting shareasecret")

	config := &shareasecret.Configuration{}

	err := config.PopulateFromEnv()
	if err != nil {
		log.Error().Err(err).Msg("populating configuration")
	}

	webAssets, err := fs.Sub(embeddedWebAssets, "web")
	if err != nil {
		log.Error().Err(err).Msg("reading embedded web/ subdir")
		os.Exit(1)
	}

	// initialize the wrapper application
	application, err := shareasecret.NewApplication(config, webAssets)
	if err != nil {
		log.Error().Err(err).Msg("initializing application")
		os.Exit(1)
	}

	// run any jobs
	application.RunDeleteExpiredSecretsJob()

	// serve all HTTP endpoints
	log.Info().Str("addr", config.Server.ListeningAddr).Msg("booting HTTP server")
	err = http.ListenAndServe(config.Server.ListeningAddr, application)
	if err != nil {
		log.Error().Err(err).Msg("listen and serve")
		os.Exit(1)
	}
}
