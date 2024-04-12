package shareasecret

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecretCreation(t *testing.T) {
	t.Run("bad request for invalid ciphertext", func(t *testing.T) {
		if r := post(t, app.handleCreateSecret, "ttl=30&encryptedSecret=a&maxViews=1"); r.statusCode != 400 {
			t.Errorf("wanted 400 status code, got %v", r.statusCode)
		} else if !strings.Contains(r.body, "format is invalid") {
			t.Errorf("wanted 'format is invalid' in body, got %v", r.body)
		}
	})

	t.Run("bad request for invalid ttl", func(t *testing.T) {
		if r := post(t, app.handleCreateSecret, "ttl=30x&encryptedSecret=a.b.c&maxViews=1"); r.statusCode != 400 {
			t.Errorf("wanted 400 status code, got %v", r.statusCode)
		} else if !strings.Contains(r.body, "parse the TTL") {
			t.Errorf("wanted 'parse the TTL' in body, got %v", r.body)
		}
	})

	t.Run("bad request for invalid maximum views", func(t *testing.T) {
		if r := post(t, app.handleCreateSecret, "ttl=30&encryptedSecret=a.b.c&maxViews=-30"); r.statusCode != 400 {
			t.Errorf("wanted 400 status code, got %v", r.statusCode)
		} else if !strings.Contains(r.body, "parse the maximum views") {
			t.Errorf("wanted 'parse the maximum views' in body, got %v", r.body)
		}
	})

	t.Run("creates the secret and redirects correctly", func(t *testing.T) {
		r := post(t, app.handleCreateSecret, "ttl=30&encryptedSecret=a.b.c&maxViews=1")
		if r.statusCode != 201 {
			t.Errorf("wanted 303 status code, got %v", r.statusCode)
		} else if _, ok := r.headers["Location"]; !ok {
			t.Errorf("expected Location response header to be present")
		}

		var rc int

		err := app.db.db.QueryRow(
			"SELECT COUNT(1) FROM secrets WHERE management_id = ?",
			strings.ReplaceAll(r.headers.Get("Location"), "/manage-secret/", ""),
		).Scan(&rc)

		if err != nil {
			t.Errorf("querying for secret: %v", err)
		} else if rc != 1 {
			t.Errorf("expected 1 secret, got %v", rc)
		}
	})
}

func post(t *testing.T, endpoint http.HandlerFunc, body string) consumedResponse {
	recorder := httptest.NewRecorder()

	r, err := http.NewRequest("POST", "anything", strings.NewReader(body))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		t.Error()
	}

	endpoint.ServeHTTP(recorder, r)

	b, err := io.ReadAll(recorder.Body)
	if err != nil {
		t.Errorf("unable to read body from endpoint (%v): %v", endpoint, err)
	}

	return consumedResponse{
		statusCode: recorder.Code,
		body:       string(b),
		headers:    recorder.Header(),
	}
}

type consumedResponse struct {
	statusCode int
	body       string
	headers    http.Header
}
