package shareasecret

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// RunDeleteExpiredSecretsJob runs a background job that identifies expired secrets and removes them accordingly
func (a *Application) RunDeleteExpiredSecretsJob() {
	runJobInBackground(
		"delete_expired_secrets",
		func(l zerolog.Logger) error {
			rows, err := a.db.db.Exec(
				`
					UPDATE
						secrets
					SET
						deleted_at = ?1,
						deletion_reason = ?2,
						cipher_text = NULL
					WHERE
						(created_at + (ttl * 60 * 1000)) <= ?1 AND
						deleted_at IS NULL
				`,
				time.Now().UnixMilli(),
				deletionReasonExpired,
			)
			if err != nil {
				return err
			}

			c, err := rows.RowsAffected()
			if err != nil {
				return err
			}

			l.Info().Int64("deleted_secrets", c).Msg("deleted expired secrets")

			return nil
		},
		1*time.Minute,
	)
}

// runJobInBackground runs the given function in a coroutine, recovering from any panics and repeating continuously,
// pausing for the specified duration after every run
func runJobInBackground(name string, f func(l zerolog.Logger) error, every time.Duration) {
	go func() {
		for {
			// this is annoying, but the only way to recover and carry on
			func() {
				l := log.With().Str("job_name", name).Logger()

				defer func() {
					if err := recover(); err != nil {
						l.Err(fmt.Errorf("recover: %v", err)).Msg("recover")
					}
				}()

				l.Debug().Msg("executing job")

				if err := f(l); err != nil {
					l.Err(err).Msg("execute")
				}

				l.Debug().Msg("executed job")

				<-time.After(every)
			}()
		}
	}()
}
