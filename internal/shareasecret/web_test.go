package shareasecret

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSecretCreation(t *testing.T) {
	t.Run("bad request for invalid ciphertext", func(t *testing.T) {
		if r := post(t, app.handleCreateSecret, "ttl=30&encryptedSecret=a&maxViews=1", emptyRequestConfigurer); r.statusCode != 400 {
			t.Errorf("wanted 400 status code, got %v", r.statusCode)
		} else if !strings.Contains(r.body, "format is invalid") {
			t.Errorf("wanted 'format is invalid' in body, got %v", r.body)
		}
	})

	t.Run("bad request for invalid ttl", func(t *testing.T) {
		if r := post(t, app.handleCreateSecret, "ttl=30x&encryptedSecret=a.b.c&maxViews=1", emptyRequestConfigurer); r.statusCode != 400 {
			t.Errorf("wanted 400 status code, got %v", r.statusCode)
		} else if !strings.Contains(r.body, "parse the TTL") {
			t.Errorf("wanted 'parse the TTL' in body, got %v", r.body)
		}
	})

	t.Run("bad request for invalid maximum views", func(t *testing.T) {
		if r := post(t, app.handleCreateSecret, "ttl=30&encryptedSecret=a.b.c&maxViews=-30", emptyRequestConfigurer); r.statusCode != 400 {
			t.Errorf("wanted 400 status code, got %v", r.statusCode)
		} else if !strings.Contains(r.body, "parse the maximum views") {
			t.Errorf("wanted 'parse the maximum views' in body, got %v", r.body)
		}
	})

	t.Run("creates the secret and redirects correctly", func(t *testing.T) {
		r := post(t, app.handleCreateSecret, "ttl=30&encryptedSecret=a.b.c&maxViews=1", emptyRequestConfigurer)
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

func TestSecretManagement(t *testing.T) {
	t.Run("redirects home if secret has been deleted", func(t *testing.T) {
		_, managementID := createSecret(t, time.Now(), deletionReasonUserDeleted)

		r := get(t, app.handleManageSecret, func(r *http.Request) { r.SetPathValue("managementID", managementID) })

		if r.statusCode != 303 {
			t.Errorf("expected 303 status code, got %v", r.statusCode)
		} else if r.headers.Get("Location") != "/" {
			t.Errorf("expected redirect to /, got %v", r.headers.Get("Location"))
		} else if c := r.cookies[0]; c.Name != "flash_err" {
			t.Errorf("expected flash_err cookie to be present")
		}
	})

	t.Run("deletes a secret", func(t *testing.T) {
		accessID, managementID := createSecret(t, time.Time{}, "")

		r := post(t, app.handleDeleteSecret, "", func(r *http.Request) { r.SetPathValue("managementID", managementID) })

		if !responseIsRedirectTo(r, "/") {
			t.Errorf("expected redirect to home page")
		} else if c := r.cookies[0]; c.Name != "flash_success" {
			t.Errorf("expected flash_success cookie to be present")
		}

		var deletedAt sql.NullInt64

		err := app.db.db.QueryRow("SELECT deleted_at FROM secrets WHERE access_id = ?", accessID).Scan(&deletedAt)
		if err != nil {
			t.Errorf("querying secret: %v", err)
		} else if !deletedAt.Valid {
			t.Errorf("expected secret's deleted_at to have been set")
		}
	})
}

func TestSecretAccess(t *testing.T) {
	t.Run("redirects home if secret has been deleted", func(t *testing.T) {
		accessID, _ := createSecret(t, time.Now(), deletionReasonUserDeleted)

		r := get(t, app.handleAccessSecretInterstitial, func(r *http.Request) { r.SetPathValue("accessID", accessID) })

		if r.statusCode != 303 {
			t.Errorf("expected 303 status code, got %v", r.statusCode)
		} else if r.headers.Get("Location") != "/" {
			t.Errorf("expected redirect to /, got %v", r.headers.Get("Location"))
		} else if c := r.cookies[0]; c.Name != "flash_err" {
			t.Errorf("expected flash_err cookie to be present")
		}
	})

	t.Run("marks secret as deleted if maximum views is reached", func(t *testing.T) {
		accessID, _ := createSecret(t, time.Time{}, "")

		r := post(t, app.handleCreateSecretView, "", func(r *http.Request) { r.SetPathValue("accessID", accessID) })
		if r.statusCode != 303 {
			t.Errorf("expected 303 status code, got %v", r.statusCode)
		} else if !strings.HasPrefix(r.headers.Get("Location"), fmt.Sprintf("/secret/%v/", accessID)) {
			t.Errorf("expected redirect to start with /secret/%v/, got %v", accessID, r.headers.Get("Location"))
		}

		r = get(t, app.handleAccessSecret, func(hr *http.Request) {
			hr.SetPathValue("accessID", accessID)
			hr.SetPathValue("viewingKey", (strings.Split(r.headers.Get("Location"), "/")[3]))
		})
		if r.statusCode != 200 {
			t.Errorf("expected 200 status code, got %v", r.statusCode)
		}

		var deletedAt sql.NullInt64
		var deletionReason sql.NullString

		err := app.db.db.
			QueryRow("SELECT deleted_at, deletion_reason FROM secrets WHERE access_id = ?", accessID).
			Scan(&deletedAt, &deletionReason)

		if err != nil {
			t.Errorf("querying secret: %v", err)
		} else if !deletedAt.Valid {
			t.Errorf("expected secret's deleted_at to have been set")
		} else if deletionReason.String != deletionReasonMaximumViewCountHit {
			t.Errorf("expected deletion reason to be maximum_view_count_hit, got %v", deletionReason.String)
		}
	})

	t.Run("returns error if secret viewing key has been used already", func(t *testing.T) {
		accessID, _ := createSecret(t, time.Time{}, "")
		viewingKey, _ := secureID(8)

		_, err := app.db.db.Exec(
			`
				INSERT INTO secret_views (secret_id, viewing_key, viewed_at, created_at)
				SELECT id, ?, NULL, ?
				FROM secrets
				WHERE access_id = ?
			`,
			viewingKey,
			time.Now().UnixMilli(),
			time.Now().UnixMilli(),
			accessID,
		)
		if err != nil {
			t.Errorf("creating secret_view: %v", err)
		}

		r := get(t, app.handleAccessSecret, func(hr *http.Request) {
			hr.SetPathValue("accessID", accessID)
			hr.SetPathValue("viewingKey", viewingKey)
		})
		if r.statusCode != 303 {
			t.Errorf("expected 303 status code, got %v", r.statusCode)
		} else if c := r.cookies[0]; c.Name != "flash_err" {
			t.Errorf("expected flash_err cookie to be present")
		}
	})
}

func post(t *testing.T, endpoint http.HandlerFunc, body string, rc func(r *http.Request)) consumedResponse {
	recorder := httptest.NewRecorder()

	r, err := http.NewRequest("POST", "anything", strings.NewReader(body))
	if err != nil {
		t.Error()
	}

	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rc(r)

	endpoint.ServeHTTP(recorder, r)

	b, err := io.ReadAll(recorder.Body)
	if err != nil {
		t.Errorf("reading body from endpoint (%v): %v", endpoint, err)
	}

	return consumedResponse{
		statusCode: recorder.Code,
		body:       string(b),
		headers:    recorder.Header(),
		cookies:    recorder.Result().Cookies(),
	}
}

var emptyRequestConfigurer = func(r *http.Request) {}

func get(t *testing.T, endpoint http.HandlerFunc, rc func(r *http.Request)) consumedResponse {
	recorder := httptest.NewRecorder()

	r, err := http.NewRequest("POST", "something", nil)
	if err != nil {
		t.Error()
	}
	rc(r)

	endpoint.ServeHTTP(recorder, r)

	b, err := io.ReadAll(recorder.Body)
	if err != nil {
		t.Errorf("reading body from endpoint (%v): %v", endpoint, err)
	}

	return consumedResponse{
		statusCode: recorder.Code,
		body:       string(b),
		headers:    recorder.Header(),
		cookies:    recorder.Result().Cookies(),
	}
}

type consumedResponse struct {
	statusCode int
	body       string
	headers    http.Header
	cookies    []*http.Cookie
}

func createSecret(t *testing.T, deletedAt time.Time, deletionReason string) (string, string) {
	accessID, _ := secureID(24)
	managementID, _ := secureID(24)

	var dbDeletedAt sql.NullInt64
	var dbDeletionReason sql.NullString

	if (deletedAt == time.Time{}) {
		dbDeletedAt = sql.NullInt64{}
		dbDeletionReason = sql.NullString{}
	} else {
		dbDeletedAt = sql.NullInt64{Valid: true, Int64: deletedAt.UnixMilli()}
		dbDeletionReason = sql.NullString{Valid: true, String: deletionReason}
	}

	_, err := app.db.db.Exec(
		`
			INSERT INTO secrets (access_id, management_id, maximum_views, ttl, cipher_text, deleted_at, deletion_reason, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`,
		accessID,
		managementID,
		1,
		30,
		"a.b.c",
		dbDeletedAt,
		dbDeletionReason,
		time.Now().UnixMilli(),
	)
	if err != nil {
		t.Errorf("creating secret: %v", err)
	}

	return accessID, managementID
}

func responseIsRedirectTo(r consumedResponse, to string) bool {
	return r.statusCode == 303 && r.headers.Get("Location") == to
}
