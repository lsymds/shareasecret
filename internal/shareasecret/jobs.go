package shareasecret

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// RunDeleteExpiredSecretsJob runs a background job that identifies expired secrets and removes them accordingly.
func (a *Application) RunDeleteExpiredSecretsJob() {
	runJobInBackground(
		"delete_expired_secrets",
		func(l zerolog.Logger) error {
			rows, err := a.db.db.Exec("DELETE FROM secrets WHERE expires_at <= ?", time.Now().UnixMilli())
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
