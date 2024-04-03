package main

import (
	"embed"
	"errors"
	"io/fs"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/lsymds/shareasecret/internal/shareasecret"
	"github.com/rs/zerolog/log"
)

//go:embed web/**
var embeddedWebAssets embed.FS

var version string = "0.0.1"

func main() {
	log.Info().Str("version", version).Msg("starting shareasecret")

	// extract any required environment variables
	err := godotenv.Load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error().Err(err).Msg("loading .env file")
		os.Exit(1)
	}

	dbPath := os.Getenv("SHAREASECRET_DB_PATH")
	if dbPath == "" {
		log.Error().Msg("SHAREASECRET_DB_PATH not set")
		os.Exit(1)
	}

	baseUrl := os.Getenv("SHAREASECRET_BASE_URL")
	if baseUrl == "" {
		log.Error().Msg("SHAREASECRET_BASE_URL not set")
		os.Exit(1)
	}

	listeningAddr := os.Getenv("SHAREASECRET_LISTENING_ADDR")
	if listeningAddr == "" {
		listeningAddr = "127.0.0.1:8994"
	}

	webAssets, err := fs.Sub(embeddedWebAssets, "web")
	if err != nil {
		log.Error().Err(err).Msg("reading embedded web/ subdir")
		os.Exit(1)
	}

	// initialize the wrapper application
	application, err := shareasecret.NewApplication("file:"+dbPath, baseUrl, webAssets)
	if err != nil {
		log.Error().Err(err).Msg("initializing application")
		os.Exit(1)
	}

	// run any jobs
	application.RunDeleteExpiredSecretsJob()

	// serve all HTTP endpoints
	log.Info().Str("addr", listeningAddr).Msg("booting HTTP server")
	err = http.ListenAndServe(listeningAddr, application)
	if err != nil {
		log.Error().Err(err).Msg("listen and serve")
		os.Exit(1)
	}
}
