package shareasecret

import (
	"os"
	"strings"
	"testing"
)

var app *Application

func TestMain(m *testing.M) {
	a, err := NewApplication("file:shareasecret_test.db", "http://127.0.0.1:8999", os.DirFS("../web/"))
	if err != nil {
		panic(err)
	}

	app = a

	defer func() {
		c, _ := os.ReadDir("./")
		for _, e := range c {
			if strings.HasPrefix(e.Name(), "shareasecret_test.db") {
				os.Remove(e.Name())
			}
		}
	}()

	m.Run()
}
