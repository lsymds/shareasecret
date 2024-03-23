package main

import (
	"net/http"

	"github.com/lsymds/shareasecret/internal/shareasecret"
)

func main() {
	application, err := shareasecret.NewApplication("file:shareasecret.db", "http://127.0.0.1:8994")
	if err != nil {
		panic(err)
	}

	// run any jobs
	application.RunDeleteExpiredSecretsJob()

	panic(http.ListenAndServe("127.0.0.1:8994", application))
}
