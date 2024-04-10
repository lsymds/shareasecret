package shareasecret

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/lsymds/staticmodtimefs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// mapRoutes maps all HTTP routes for the application.
func (a *Application) mapRoutes() {
	fs := http.FileServerFS(staticmodtimefs.NewStaticModTimeFS(a.webAssets, time.Now()))
	a.router.Handle("GET /static/", http.StripPrefix("/static/", fs))
	a.router.Handle("GET /robots.txt", a.serveFile("robots.txt"))

	a.router.HandleFunc("GET /", a.handleGetIndex)

	a.router.Handle("GET /nojs", templ.Handler(pageNoJavascript()))
	a.router.Handle("GET /oops", templ.Handler(pageOops()))

	a.router.HandleFunc("POST /secret", a.handleCreateSecret)
	a.router.HandleFunc("GET /secret/{accessID}", a.handleAccessSecretInterstitial)
	a.router.HandleFunc("POST /secret/{accessID}", a.handleCreateSecretView)
	a.router.HandleFunc("GET /secret/{accessID}/{viewingKey}", a.handleAccessSecret)
	a.router.HandleFunc("GET /manage-secret/{managementID}", a.handleManageSecret)
	a.router.HandleFunc("POST /manage-secret/{managementID}/delete", a.handleDeleteSecret)
}

func (a *Application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	loggingMiddleware(
		recoveryMiddleware(
			a.router,
		),
	).ServeHTTP(w, r)
}

func recoveryMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := zerolog.Ctx(r.Context())

		defer func() {
			if err := recover(); err != nil {
				l.Error().Any("panic", err).Msg("recovered from panic")
				http.Redirect(w, r, "/oops", http.StatusSeeOther)
			}
		}()

		h.ServeHTTP(w, r)
	})
}

func loggingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := log.With().
			Str("url", r.URL.String()).
			Str("method", r.Method)

		r = r.WithContext(l.Logger().WithContext(r.Context()))

		h.ServeHTTP(w, r)
	})
}

func (a *Application) serveFile(fileName string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, a.webAssets, fileName)
	})
}

func (a *Application) handleGetIndex(w http.ResponseWriter, r *http.Request) {
	pageIndex(notificationsFromRequest(r, w)).Render(r.Context(), w)
}

func (a *Application) handleCreateSecret(w http.ResponseWriter, r *http.Request) {
	l := zerolog.Ctx(r.Context())

	secret := ""
	ttl := 0
	maxViews := 0

	// parse and validate the request
	if err := r.ParseForm(); err != nil {
		badRequest("Unable to parse request form. Please try again.", w)
		return
	} else {
		// very little we can do here aside from validating the structure of the "encrypted" text string received matches
		// how the front-end should have formatted it
		secret = r.Form.Get("encryptedSecret")
		if strings.Count(secret, ".") != 2 {
			badRequest("Secret format is invalid. Please try again.", w)
			return
		}

		ttl, err = strconv.Atoi(r.Form.Get("ttl"))
		if err != nil {
			badRequest("Unable to parse the TTL (time to live) for the secret.", w)
			return
		}

		maxViews, err = strconv.Atoi(r.Form.Get("maxViews"))
		if err != nil || maxViews < 0 {
			badRequest("Unable to parse the maximum views permitted for the secret.", w)
			return
		}
	}

	// create the secret, and generate two cryptographically random, 192 bit identifiers to use for viewing and
	// management of the secret respectively
	accessID, err := secureID(24)
	if err != nil {
		l.Err(err).Msg("generating access id")
		internalServerError(w)
		return
	}

	managementID, err := secureID(24)
	if err != nil {
		l.Err(err).Msg("generating management id")
		internalServerError(w)
		return
	}

	if _, err := a.db.db.Exec(
		`
			INSERT INTO
				secrets (access_id, management_id, cipher_text, ttl, maximum_views, created_at)
			VALUES
				(?, ?, ?, ?, ?, ?)
		`,
		accessID,
		managementID,
		secret,
		ttl,
		maxViews,
		time.Now().UnixMilli(),
	); err != nil {
		l.Err(err).Msg("creating secret")
		internalServerError(w)
		return
	}

	// redirect the user to the manage secrets page
	http.Redirect(w, r, fmt.Sprintf("/manage-secret/%s", managementID), http.StatusCreated)
}

func (a *Application) handleAccessSecretInterstitial(w http.ResponseWriter, r *http.Request) {
	accessID := r.PathValue("accessID")

	l := zerolog.Ctx(r.Context()).
		With().
		Str("access_id", accessID).
		Logger()

	// retrieve the row identifier of the secret if it exists and has not been deleted
	var secretID int
	err := a.db.db.QueryRow(
		`
			SELECT
				id
			FROM
				secrets
			WHERE
				access_id = ? AND
				deleted_at IS NULL
		`,
		accessID,
	).Scan(&secretID)

	if errors.Is(sql.ErrNoRows, err) {
		setFlashErr("Secret does not exist or has been deleted.", w)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	} else if err != nil {
		l.Err(err).Msg("retrieving secret")
		redirectToOopsPage(w, r)
		return
	}

	pageViewSecretInterstitial().Render(r.Context(), w)
}

func (a *Application) handleCreateSecretView(w http.ResponseWriter, r *http.Request) {
	accessID := r.PathValue("accessID")

	l := zerolog.Ctx(r.Context()).
		With().
		Str("access_id", accessID).
		Logger()

	// create a 64 bit viewing key for the secret view record
	key, err := secureID(8)
	if err != nil {
		l.Err(err).Msg("creating secret viewing key")
		redirectToOopsPage(w, r)
		return
	}

	// create the secret view without a viewing date
	rs, err := a.db.db.Exec(
		`
			INSERT INTO secret_views (secret_id, viewing_key, created_at)
			SELECT
				id,
				?,
				?
			FROM
				secrets
			WHERE
				access_id = ? AND
				deleted_at IS NULL
		`,
		key,
		time.Now().UnixMilli(),
		accessID,
	)

	if errors.Is(sql.ErrNoRows, err) {
		setFlashErr("Secret does not exist or has been deleted.", w)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	} else if err != nil {
		l.Err(err).Msg("creating secret view")
		redirectToOopsPage(w, r)
		return
	} else if rc, err := rs.RowsAffected(); err != nil || rc == 0 {
		l.Error().Err(err).Msg("creating secret view")
		redirectToOopsPage(w, r)
		return
	}

	// redirect them to the actual viewing page of the secret
	http.Redirect(w, r, fmt.Sprintf("/secret/%s/%s", accessID, key), http.StatusSeeOther)
}

func (a *Application) handleAccessSecret(w http.ResponseWriter, r *http.Request) {
	accessID := r.PathValue("accessID")
	viewingKey := r.PathValue("viewingKey")

	notifications := notifications{}

	l := zerolog.
		Ctx(r.Context()).
		With().
		Str("access_id", accessID).
		Str("viewing_key", viewingKey).
		Logger()

	// begin a transaction so the retrieval of the secret's details and the recording of the view being used are atomic
	tx, err := a.db.db.Begin()
	if err != nil {
		l.Err(err).Msg("begin tx")
		redirectToOopsPage(w, r)
		return
	}

	defer tx.Rollback()

	// retrieve the cipher text and secret view id for the relevant secret, or return an error if that secret cannot be found
	var cipherText string
	var secretViewID int
	var maxViews int
	var currentViews int

	err = tx.QueryRow(
		`
			SELECT
				s.cipher_text,
				v.id,
				s.maximum_views,
				(SELECT COUNT(1) FROM secret_views v2 WHERE v2.secret_id = v.secret_id AND viewed_at IS NOT NULL)
			FROM
				secrets s
				INNER JOIN secret_views v ON v.secret_id = s.id
			WHERE
				s.access_id = ? AND
				s.deleted_at IS NULL AND
				v.viewing_key = ? AND
				v.viewed_at IS NULL
		`,
		accessID,
		viewingKey,
	).Scan(&cipherText, &secretViewID, &maxViews, &currentViews)

	if errors.Is(sql.ErrNoRows, err) {
		setFlashErr("Secret does not exist, has been deleted, or the unique viewing key you attempted to use has been used before.", w)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	} else if err != nil {
		l.Err(err).Msg("retrieving secret")
		redirectToOopsPage(w, r)
		return
	}

	// record the secret view as being used so nobody else can use it to see the secret
	_, err = tx.Exec("UPDATE secret_views SET viewed_at = ? WHERE id = ?", time.Now().UnixMilli(), secretViewID)
	if err != nil {
		l.Err(err).Msg("updating secret view")
		redirectToOopsPage(w, r)
		return
	}

	// mark the secret as being deleted if this view is equal to or exceeds the maximum permitted views for the secret
	if maxViews > 0 && currentViews+1 >= maxViews {
		_, err := tx.Exec(
			"UPDATE secrets SET deleted_at = ?, deletion_reason = ?, cipher_text = NULL WHERE access_id = ?",
			time.Now().UnixMilli(),
			deletionReasonMaximumViewCountHit,
			accessID,
		)

		if err != nil {
			l.Err(err).Msg("deleting secret")
			redirectToOopsPage(w, r)
			return
		}

		notifications.warningMsg = "Maximum views reached. This secret will not be accessible again."
	}

	// commit the transaction
	err = tx.Commit()
	if err != nil {
		l.Err(err).Msg("committing tx")
		redirectToOopsPage(w, r)
		return
	}

	pageViewSecret(cipherText, notifications).Render(r.Context(), w)
}

func (a *Application) handleManageSecret(w http.ResponseWriter, r *http.Request) {
	managementID := r.PathValue("managementID")

	l := zerolog.
		Ctx(r.Context()).
		With().
		Str("management_id", managementID).
		Logger()

	// retrieve the ID in order to view and decrypt the secret, or return an error if that secret cannot be found
	var accessID string

	err := a.db.db.QueryRow(
		`
			SELECT
				access_id
			FROM
				secrets
			WHERE
				management_id = ? AND
				deleted_at IS NULL
		`,
		managementID,
	).Scan(&accessID)

	if errors.Is(sql.ErrNoRows, err) {
		setFlashErr("Secret does not exist or has been deleted.", w)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	} else if err != nil {
		l.Err(err).Msg("retrieving secret")
		redirectToOopsPage(w, r)
		return
	}

	pageManageSecret(
		fmt.Sprintf("%s/secret/%s", a.baseURL, accessID),
		fmt.Sprintf("%s/manage-secret/%s/delete", a.baseURL, managementID),
		notificationsFromRequest(r, w),
	).Render(r.Context(), w)
}

func (a *Application) handleDeleteSecret(w http.ResponseWriter, r *http.Request) {
	l := zerolog.Ctx(r.Context())
	managementID := r.PathValue("managementID")

	// delete the secret, returning the user to the manage secret page with an error message if that fails
	_, err := a.db.db.Exec(
		"UPDATE secrets SET deleted_at = ?, deletion_reason = ?, cipher_text = NULL WHERE management_id = ?",
		time.Now().UnixMilli(),
		deletionReasonUserDeleted,
		managementID,
	)
	if err != nil {
		l.Err(err).Str("management_id", managementID).Msg("deleting secret")
		redirectToOopsPage(w, r)
		return
	}

	setFlashSuccess("Secret successfully deleted.", w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func badRequest(err string, w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(err))
}

func internalServerError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
}

func redirectToOopsPage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/oops", http.StatusSeeOther)
}

func setFlashErr(msg string, w http.ResponseWriter) {
	setFlash("err", msg, w)
}

func setFlashWarning(msg string, w http.ResponseWriter) {
	setFlash("warn", msg, w)
}

func setFlashSuccess(msg string, w http.ResponseWriter) {
	setFlash("success", msg, w)
}

func setFlash(name string, msg string, w http.ResponseWriter) {
	n := fmt.Sprintf("flash_%s", name)
	m := base64.StdEncoding.EncodeToString([]byte(msg))
	http.SetCookie(w, &http.Cookie{Name: n, Value: m, Path: "/", HttpOnly: true})
}

func notificationsFromRequest(r *http.Request, w http.ResponseWriter) notifications {
	return notifications{
		errorMsg:   flash("err", r, w),
		warningMsg: flash("warn", r, w),
		successMsg: flash("success", r, w),
	}
}

func flash(name string, r *http.Request, w http.ResponseWriter) string {
	n := fmt.Sprintf("flash_%s", name)

	// read the cookie, returning an empty string if it doesn't exist
	c, err := r.Cookie(n)
	if err != nil {
		return ""
	}

	// set a cookie with the same name so it is "expired" within the client's browser
	http.SetCookie(
		w,
		&http.Cookie{
			Name:     n,
			Value:    "",
			Expires:  time.Unix(1, 0),
			MaxAge:   -1,
			Path:     "/",
			HttpOnly: true,
		},
	)

	// extract the base64 encoded cookie value and decode it before returning it
	v, err := base64.StdEncoding.DecodeString(c.Value)
	if err != nil {
		return ""
	}

	return string(v)
}

func secureID(size int) (string, error) {
	b := make([]byte, size)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}
