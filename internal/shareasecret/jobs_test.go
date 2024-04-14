package shareasecret

import (
	"database/sql"
	"testing"
	"time"
)

func TestDeleteExpiredSecretsJob(t *testing.T) {
	t.Run("deletes secret that should have expired", func(t *testing.T) {
		accessID, _ := createSecret(t, time.Time{}, "")

		_, err := app.db.db.Exec(
			"UPDATE secrets SET ttl = 1, created_at = ? WHERE access_id = ?",
			time.Now().Add(-2*time.Minute),
			accessID,
		)
		if err != nil {
			t.Errorf("updating secret TTL: %v", err)
		}

		app.RunDeleteExpiredSecretsJob()

		until(
			t,
			func() bool {
				var deletedAt sql.NullInt64

				err := app.db.db.QueryRow("SELECT deleted_at FROM secrets WHERE access_id = ?", accessID).Scan(&deletedAt)
				if err != nil {
					t.Errorf("querying secret: %v", err)
				}

				return deletedAt.Valid
			},
			10,
			5*time.Millisecond,
		)
	})
}
