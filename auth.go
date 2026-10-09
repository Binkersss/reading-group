package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
)

var (
	adminHash  [32]byte
	adminToken string
)

func voterID(w http.ResponseWriter, r *http.Request) string {
	if c, err := r.Cookie("voter"); err == nil && len(c.Value) == 32 {
		return c.Value
	}
	b := make([]byte, 16)
	rand.Read(b)
	id := hex.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{
		Name: "voter", Value: id, Path: "/", MaxAge: 60 * 60 * 24 * 365 * 2,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: isHTTPS(r),
	})
	return id
}

func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func isAdmin(r *http.Request) bool {
	c, err := r.Cookie("admin")
	return err == nil && subtle.ConstantTimeCompare([]byte(c.Value), []byte(adminToken)) == 1
}

func requireAdmin(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			if r.Header.Get("HX-Request") != "" {
				w.Header().Set("HX-Redirect", "/admin/login")
				return
			}
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		h(w, r)
	}
}
