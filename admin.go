package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func loginPage(w http.ResponseWriter, r *http.Request) {
	if isAdmin(r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	render(w, "admin_login", "")
}

func login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", 400)
		return
	}
	given := sha256.Sum256([]byte(r.FormValue("password")))
	if subtle.ConstantTimeCompare(given[:], adminHash[:]) != 1 {
		time.Sleep(500 * time.Millisecond) // blunt brute-force speed bump
		w.WriteHeader(http.StatusUnauthorized)
		render(w, "admin_login", "Wrong password.")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: "admin", Value: adminToken, Path: "/", MaxAge: 60 * 60 * 24 * 14,
		HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: isHTTPS(r),
	})
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "admin", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ---------- admin side ----------

func adminHome(w http.ResponseWriter, r *http.Request) {
	list, err := listPolls()
	if err != nil {
		http.Error(w, "db error", 500)
		return
	}
	render(w, "admin_home", list)
}

func adminCreate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", 400)
		return
	}
	q := clip(r.FormValue("question"), maxQuestions)

	var labels []string
	seen := map[string]bool{}
	for _, line := range strings.Split(r.FormValue("options"), "\n") {
		l := clip(line, maxLabel)
		if l == "" || seen[strings.ToLower(l)] {
			continue
		}
		seen[strings.ToLower(l)] = true
		labels = append(labels, l)
	}

	switch {
	case q == "":
		render(w, "err", "Give the poll a question.")
		return
	case len(labels) < 2:
		render(w, "err", "Add at least two options, one per line.")
		return
	case len(labels) > maxOptions:
		render(w, "err", "At most "+strconv.Itoa(maxOptions)+" options.")
		return
	}
	allow := 0
	if r.FormValue("allow_suggest") == "1" {
		allow = 1
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "db error", 500)
		return
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO polls(question, allow_suggest) VALUES (?, ?)`, q, allow)
	if err != nil {
		http.Error(w, "db error", 500)
		return
	}
	id, _ := res.LastInsertId()
	for _, l := range labels {
		if _, err := tx.Exec(`INSERT INTO options(poll_id, label) VALUES (?, ?)`, id, l); err != nil {
			http.Error(w, "db error", 500)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "db error", 500)
		return
	}
	w.Header().Set("HX-Redirect", "/admin/polls/"+strconv.FormatInt(id, 10))
}

func renderAdminBody(w http.ResponseWriter, r *http.Request, id int64) {
	p, err := loadAdminPoll(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	render(w, "admin_body", p)
}

func adminPoll(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	p, err := loadAdminPoll(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	render(w, "admin_poll", p)
}

func adminPanel(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	p, err := loadAdminPoll(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	render(w, "panel", p)
}

// adminSet runs one UPDATE against a poll and re-renders the admin body.
func adminSet(query string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r, "id")
		if !ok {
			http.NotFound(w, r)
			return
		}
		if _, err := db.Exec(query, id); err != nil {
			http.Error(w, "db error", 500)
			return
		}
		renderAdminBody(w, r, id)
	}
}

func adminAddOption(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if label := clip(r.FormValue("label"), maxLabel); label != "" {
		if err := addOption(id, label, false); err != nil {
			http.Error(w, "db error", 500)
			return
		}
	}
	renderAdminBody(w, r, id)
}

func adminDeleteOption(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	oid, ok2 := pathID(r, "oid")
	if !ok || !ok2 {
		http.NotFound(w, r)
		return
	}
	// Votes for the option go with it (ON DELETE CASCADE).
	if _, err := db.Exec(`DELETE FROM options WHERE id = ? AND poll_id = ?`, oid, id); err != nil {
		http.Error(w, "db error", 500)
		return
	}
	renderAdminBody(w, r, id)
}

func adminDeletePoll(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if _, err := db.Exec(`DELETE FROM polls WHERE id = ?`, id); err != nil {
		http.Error(w, "db error", 500)
		return
	}
	w.Header().Set("HX-Redirect", "/admin")
}
