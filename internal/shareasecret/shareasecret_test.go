package shareasecret

import (
	"database/sql"
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

// createSecret creates a secret instance in the database
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

// until continuously loops until the given function returns truthy or the maximum tries are exceeded (at which point a
// test failure will occur)
func until(t *testing.T, try func() bool, maximumTries uint8, delay time.Duration) {
	for i := 0; i < int(maximumTries); i++ {
		if r := try(); r {
			return
		}

		<-time.After(delay)
	}

	t.Error("until maximumTries exceeded")
}
