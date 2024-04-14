package shareasecret

import (
	"os"
	"strings"
	"testing"
	"time"
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

func until(t *testing.T, try func() bool, maximumTries uint8, delay time.Duration) {
	for i := 0; i < int(maximumTries); i++ {
		if r := try(); r {
			return
		}

		<-time.After(delay)
	}

	t.Error("until maximumTries exceeded")
}
