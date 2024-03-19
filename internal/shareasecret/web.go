package shareasecret

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// mapRoutes maps all HTTP routes for the application.
func (a *Application) mapRoutes() {
	fs := http.FileServer(http.Dir("./static/"))
	a.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	a.router.HandleFunc("/", a.handleGetIndex).Methods("GET")
	a.router.HandleFunc("/nojs", a.handleNoJavascriptNotice).Methods("GET")
	a.router.HandleFunc("/secret", a.handleCreateSecret).Methods("POST")
	a.router.HandleFunc("/secret/{secretId}", a.handleGetSecret).Methods("GET")
	a.router.HandleFunc("/manage-secret/{managementId}", a.handleManageSecret).Methods("GET")
	a.router.HandleFunc("/manage-secret/{managementId}", a.handleDeleteSecret).Methods("DELETE")
}

func (a *Application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

func (a *Application) handleGetIndex(w http.ResponseWriter, r *http.Request) {
	pageIndex().Render(r.Context(), w)
}

func (a *Application) handleNoJavascriptNotice(w http.ResponseWriter, r *http.Request) {
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

	// create the secret, and generate two cryptographically random, 256 bit identifiers to use for viewing and
	// management of the secret respectively
	viewingId, err := randomSecureId()
	if err != nil {
		internalServerError("Unable to create the secret. Please try again.", w)
		return
	}

	managingId, err := randomSecureId()
	if err != nil {
		internalServerError("Unable to create the secret. Please try again.", w)
		return
	}

	if _, err := a.db.db.Exec(
		`
			INSERT INTO secrets (view_id, manage_id, cipher_text, ttl, alive_until, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`,
		viewingId,
		managingId,
		secret,
		ttl,
		time.Now().Add(time.Duration(ttl)*time.Minute).UnixMilli(),
		time.Now().UnixMilli(),
	); err != nil {
		internalServerError("Unable to create the secret. Please try again.", w)
		return
	}

	// redirect the user to the manage secrets page
	http.Redirect(w, r, fmt.Sprintf("/manage-secret/%s", managingId), http.StatusCreated)
}

func (a *Application) handleGetSecret(w http.ResponseWriter, r *http.Request) {}

func (a *Application) handleManageSecret(w http.ResponseWriter, r *http.Request) {}

func (a *Application) handleDeleteSecret(w http.ResponseWriter, r *http.Request) {}

func badRequest(err string, w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(err))
}

func internalServerError(err string, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err))
}

func randomSecureId() (string, error) {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}
