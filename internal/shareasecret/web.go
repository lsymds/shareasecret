package shareasecret

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/lsymds/go-utils/pkg/http/middleware"
	"github.com/lsymds/staticmodtimefs"
	"github.com/rs/zerolog"
)

// mapRoutes maps all HTTP routes for the application.
func (a *Application) mapRoutes() {
	assetsFS := staticmodtimefs.NewStaticModTimeFS(a.webAssets, time.Now())
	assetsFSHandler := http.FileServerFS(assetsFS)

	a.router.Handle("GET /static/", http.StripPrefix("/static/", assetsFSHandler))
	a.router.Handle("GET /robots.txt", serveFile(assetsFS, "robots.txt"))

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

// ServeHTTP is the root [http.Handler] method for the application. It serves all application routes, wrapping them with
// any required middlewares
func (a *Application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	middleware.Logging(
		middleware.Recovery(
			a.router,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/oops", http.StatusSeeOther)
			}),
		),
		nil,
	).ServeHTTP(w, r)
}

// serveFile serves an individual file over HTTP from a filesystem
func serveFile(fs fs.FS, fileName string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, fs, fileName)
	})
}

// handleGetIndex renders the root page for the application - this is where visitors are able to create secrets
// (performed in the [handleCreateSecret] handler)
func (a *Application) handleGetIndex(w http.ResponseWriter, r *http.Request) {
	ns := notificationsFromRequest(r, w)
	ipRestricted := !requestingIPCanCreateSecret(a.config, r)

	pageIndex(ns, ipRestricted).Render(r.Context(), w)
}

// handleCreateSecret validates and persists a secret (consisting of encrypted ciphertext)
func (a *Application) handleCreateSecret(w http.ResponseWriter, r *http.Request) {
	l := zerolog.Ctx(r.Context())

	// redirect to the home page if requester is not permitted to create secrets
	if !requestingIPCanCreateSecret(a.config, r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

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

	http.Redirect(w, r, fmt.Sprintf("/manage-secret/%s", managementID), http.StatusCreated)
}

// handleAccessSecretInterstitial presents a disclaimer to the visitor informing them that proceeding will use
// a 'view' of the secret
//
// Accessing this page by itself does not constitute a view or modify a secret in anyway.
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

// handleCreateSecretView creates a 'view' of a secret and is the POST accompaniment to the
// [handleAccessSecretInterstitial] handler.
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

	// create the secret view without a viewing date, as this will be set when the viewing page route is actually
	// called
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

	// redirect them to the actual viewing page of the secret (which will then mark the secret view as viewed)
	http.Redirect(w, r, fmt.Sprintf("/secret/%s/%s", accessID, key), http.StatusSeeOther)
}

// handleAccessSecret serves the 'decryption' page for a secret providing that a valid access identifier (192 bit) and
// access key (32 bit) are provided in the request
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

	// retrieve the cipher text and secret view id for the relevant secret, or return an error if that secret cannot be
	// found
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

	err = tx.Commit()
	if err != nil {
		l.Err(err).Msg("committing tx")
		redirectToOopsPage(w, r)
		return
	}

	pageViewSecret(cipherText, notifications).Render(r.Context(), w)
}

// handleManageSecret renders the management page of a secret and is intended for the original creator of the secret
// to view
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

// handleDeleteSecret deletes a secret
func (a *Application) handleDeleteSecret(w http.ResponseWriter, r *http.Request) {
	l := zerolog.Ctx(r.Context())
	managementID := r.PathValue("managementID")

	// delete the secret (if it hasn't already been deleted), returning the user to the manage secret page with an error
	// message if that fails
	_, err := a.db.db.Exec(
		"UPDATE secrets SET deleted_at = ?, deletion_reason = ?, cipher_text = NULL WHERE management_id = ? AND deleted_at IS NULL",
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

// badRequest sets the status code of the response to 400 and writes the error to the body
func badRequest(err string, w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(err))
}

// internalServerError sets the status code of the response to 500
func internalServerError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
}

// redirectToOopsPage configures the response to redirect to the /oops route which is a catch all error page for
// any errors that weren't expected
func redirectToOopsPage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/oops", http.StatusSeeOther)
}

// setFlashErr sets a flash cookie for errors with the content provided
func setFlashErr(msg string, w http.ResponseWriter) {
	setFlash("err", msg, w)
}

// setFlashSuccess sets a flash cookie for successes with the content provided
func setFlashSuccess(msg string, w http.ResponseWriter) {
	setFlash("success", msg, w)
}

// setFlash sets a flash cookie of the given name and message
func setFlash(name string, msg string, w http.ResponseWriter) {
	n := fmt.Sprintf("flash_%s", name)
	m := base64.StdEncoding.EncodeToString([]byte(msg))
	http.SetCookie(w, &http.Cookie{Name: n, Value: m, Path: "/", HttpOnly: true})
}

// notificationsFromRequest extracts a [notifications] instance from any flash cookies in the request
func notificationsFromRequest(r *http.Request, w http.ResponseWriter) notifications {
	return notifications{
		errorMsg:   flash("err", r, w),
		warningMsg: flash("warn", r, w),
		successMsg: flash("success", r, w),
	}
}

// flash extracts a given flash message cookie from the request
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

// secureID generates a randomised hexadecimal identifier of the size in bytes from a secure cryptorandom source
func secureID(size int) (string, error) {
	b := make([]byte, size)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

// requestingIPCanCreateSecret identifies whether the request was made from an IP address that has been specifically
// allowed to create secrets.
func requestingIPCanCreateSecret(config *Configuration, r *http.Request) bool {
	if len(config.SecretCreationRestrictions.IPAddresses.FixedIPs) == 0 && len(config.SecretCreationRestrictions.IPAddresses.CIDRs) == 0 {
		return true
	}

	sourceIP := net.ParseIP(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
	if sourceIP == nil {
		return false
	}

	// check if any IPs match directly
	for _, ip := range config.SecretCreationRestrictions.IPAddresses.FixedIPs {
		if ip.Equal(sourceIP) {
			return true
		}
	}

	// check if any specified CIDRs contain the ip address
	for _, cidr := range config.SecretCreationRestrictions.IPAddresses.CIDRs {
		if cidr.Contains(sourceIP) {
			return true
		}
	}

	return false
}
