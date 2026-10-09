// Reading group polls: one file, Go + htmx + SQLite.
//
//	ADMIN_PASSWORD=secret go run .      # http://localhost:8080
//
// User side:  /            browse polls, vote, suggest options (if the admin allows it)
// Admin side: /admin       create polls, manage options, open/close, delete
//
// Env: ADMIN_PASSWORD (random one is generated and logged if unset),
//
//	ADDR (default :8080), DB (default polls.db)
package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log"
	"net/http"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	pw := os.Getenv("ADMIN_PASSWORD")
	if pw == "" {
		b := make([]byte, 6)
		rand.Read(b)
		pw = hex.EncodeToString(b)
		log.Printf("ADMIN_PASSWORD not set; generated one for this run: %s", pw)
	}
	adminHash = sha256.Sum256([]byte(pw))
	mac := hmac.New(sha256.New, adminHash[:])
	mac.Write([]byte("reading-polls-admin-session"))
	adminToken = hex.EncodeToString(mac.Sum(nil))

	path := os.Getenv("DB")
	if path == "" {
		path = "polls.db"
	}
	var err error
	db, err = sql.Open("sqlite",
		"file:"+path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(1) // SQLite: serialize writers, keeps things simple
	if _, err := db.Exec(schema); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	fileServerStatic := http.FileServer(http.Dir("static"))

	// user side
	mux.HandleFunc("GET /{$}", home)
	mux.Handle("/static/", http.StripPrefix("/static/", fileServerStatic))
	mux.HandleFunc("GET /polls/{id}", showPoll)
	mux.HandleFunc("GET /polls/{id}/panel", panel)
	mux.HandleFunc("POST /polls/{id}/vote", vote)
	mux.HandleFunc("POST /polls/{id}/options", suggest)

	// admin side
	mux.HandleFunc("GET /admin/login", loginPage)
	mux.HandleFunc("POST /admin/login", login)
	mux.HandleFunc("POST /admin/logout", logout)
	mux.HandleFunc("GET /admin", requireAdmin(adminHome))
	mux.HandleFunc("POST /admin/polls", requireAdmin(adminCreate))
	mux.HandleFunc("GET /admin/polls/{id}", requireAdmin(adminPoll))
	mux.HandleFunc("GET /admin/polls/{id}/panel", requireAdmin(adminPanel))
	mux.HandleFunc("POST /admin/polls/{id}/close", requireAdmin(adminSet(`UPDATE polls SET closed = 1 WHERE id = ?`)))
	mux.HandleFunc("POST /admin/polls/{id}/reopen", requireAdmin(adminSet(`UPDATE polls SET closed = 0 WHERE id = ?`)))
	mux.HandleFunc("POST /admin/polls/{id}/suggestions", requireAdmin(adminSet(`UPDATE polls SET allow_suggest = 1 - allow_suggest WHERE id = ?`)))
	mux.HandleFunc("POST /admin/polls/{id}/options", requireAdmin(adminAddOption))
	mux.HandleFunc("POST /admin/polls/{id}/options/{oid}/delete", requireAdmin(adminDeleteOption))
	mux.HandleFunc("POST /admin/polls/{id}/delete", requireAdmin(adminDeletePoll))

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("listening on %s (db: %s)", addr, path)
	log.Fatal(http.ListenAndServe(addr, mux))
}
