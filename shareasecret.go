package main

import (
	"net/http"

	"github.com/lsymds/shareasecret/internal/shareasecret"
)

func main() {
	application, err := shareasecret.NewApplication("file:foobar?mode=memory&cache=shared")
	if err != nil {
		panic(err)
	}

	http.ListenAndServe("127.0.0.1:8994", application)
}
