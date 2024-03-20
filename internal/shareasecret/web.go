package shareasecret

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/gorilla/mux"
)

// mapRoutes maps all HTTP routes for the application.
func (a *Application) mapRoutes() {
	fs := http.FileServer(http.Dir("./static/"))
	a.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	a.router.HandleFunc("/", a.handleGetIndex).Methods("GET")

	a.router.Handle("/nojs", templ.Handler(pageNoJavascript())).Methods("GET")
	a.router.Handle("/oops", templ.Handler(pageOops())).Methods("GET")

	a.router.HandleFunc("/secret", a.handleCreateSecret).Methods("POST")
	a.router.HandleFunc("/secret/{viewingID}", a.handleGetSecret).Methods("GET")
	a.router.HandleFunc("/manage-secret/{managementID}", a.handleManageSecret).Methods("GET")
	a.router.HandleFunc("/manage-secret/{managementID}/delete", a.handleDeleteSecret).Methods("POST")
}

func (a *Application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

func (a *Application) handleGetIndex(w http.ResponseWriter, r *http.Request) {
	pageIndex(notificationsFromRequest(r, w)).Render(r.Context(), w)
}

func (a *Application) handleCreateSecret(w http.ResponseWriter, r *http.Request) {
	secret := ""
	ttl := 0

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
	}

	// create the secret, and generate two cryptographically random, 192 bit identifiers to use for viewing and
	// management of the secret respectively
	viewingID, err := secureID()
	if err != nil {
		internalServerError("Unable to create the secret. Please try again.", w)
		return
	}

	managementID, err := secureID()
	if err != nil {
		internalServerError("Unable to create the secret. Please try again.", w)
		return
	}

	if _, err := a.db.db.Exec(
		`
			INSERT INTO
				secrets (viewing_id, management_id, cipher_text, ttl, expires_at, created_at)
			VALUES
				(?, ?, ?, ?, ?, ?)
		`,
		viewingID,
		managementID,
		secret,
		ttl,
		time.Now().Add(time.Duration(ttl)*time.Minute).UnixMilli(),
		time.Now().UnixMilli(),
	); err != nil {
		internalServerError("Unable to create the secret. Please try again.", w)
		return
	}

	// redirect the user to the manage secrets page
	http.Redirect(w, r, fmt.Sprintf("/manage-secret/%s", managementID), http.StatusCreated)
}

func (a *Application) handleGetSecret(w http.ResponseWriter, r *http.Request) {
	viewingID := mux.Vars(r)["viewingID"]

	// retrieve the cipher text for the relevant secret, or return an error if that secret cannot be found
	var cipherText string

	err := a.db.db.QueryRow(
		`
			SELECT
				cipher_text
			FROM
				secrets
			WHERE
				viewing_id = ? AND
				expires_at > ?
		`,
		viewingID,
		time.Now().UnixMilli(),
	).Scan(&cipherText)

	if errors.Is(sql.ErrNoRows, err) {
		setFlashErr("Secret does not exist or has been deleted.", w)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	} else if err != nil {
		http.Redirect(w, r, "/oops", http.StatusSeeOther)
		return
	}

	pageViewSecret(cipherText, notificationsFromRequest(r, w)).Render(r.Context(), w)
}

func (a *Application) handleManageSecret(w http.ResponseWriter, r *http.Request) {
	managementID := mux.Vars(r)["managementID"]

	// retrieve the ID in order to view and decrypt the secret, or return an error if that secret cannot be found
	var secretID string

	err := a.db.db.QueryRow(
		`
			SELECT
				viewing_id
			FROM
				secrets
			WHERE
				management_id = ? AND
				expires_at > ?
		`,
		managementID,
		time.Now().UnixMilli(),
	).Scan(&secretID)

	if errors.Is(sql.ErrNoRows, err) {
		setFlashErr("Secret does not exist or has been deleted.", w)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	} else if err != nil {
		http.Redirect(w, r, "/oops", http.StatusSeeOther)
		return
	}

	pageManageSecret(
		managementID,
		fmt.Sprintf("%s/secret/%s", a.baseURL, secretID),
		fmt.Sprintf("%s/manage-secret/%s/delete", a.baseURL, managementID),
		notificationsFromRequest(r, w),
	).Render(r.Context(), w)
}

func (a *Application) handleDeleteSecret(w http.ResponseWriter, r *http.Request) {
	managementID := mux.Vars(r)["managementID"]

	// delete the secret, returning the user to the manage secret page with an error message if that fails
	_, err := a.db.db.Exec("DELETE FROM secrets WHERE management_id = ?", managementID)
	if errors.Is(sql.ErrNoRows, err) {
		setFlashErr("Secret does not exist or has been deleted.", w)
		http.Redirect(w, r, fmt.Sprintf("/manage-secret/%s", managementID), http.StatusSeeOther)
		return
	} else if err != nil {
		http.Redirect(w, r, "/oops", http.StatusSeeOther)
		return
	}

	setFlashSuccess("Secret successfully deleted.", w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func badRequest(err string, w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(err))
}

func internalServerError(err string, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err))
}

func setFlashErr(msg string, w http.ResponseWriter) {
	setFlash("err", msg, w)
}

func setFlashSuccess(msg string, w http.ResponseWriter) {
	setFlash("success", msg, w)
}

func setFlash(name string, msg string, w http.ResponseWriter) {
	n := fmt.Sprintf("flash_%s", name)
	http.SetCookie(w, &http.Cookie{Name: n, Value: msg, Path: "/"})
}

func notificationsFromRequest(r *http.Request, w http.ResponseWriter) notifications {
	return notifications{
		errorMsg:   flash("err", r, w),
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
			Name:    n,
			Value:   "",
			Expires: time.Unix(1, 0),
			MaxAge:  -1,
			Path:    "/",
		},
	)

	return c.Value
}

func secureID() (string, error) {
	b := make([]byte, 24)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}
