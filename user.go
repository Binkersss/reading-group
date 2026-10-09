package main

import (
	"net/http"
	"strconv"
)

func home(w http.ResponseWriter, r *http.Request) {
	voterID(w, r)
	list, err := listPolls()
	if err != nil {
		http.Error(w, "db error", 500)
		return
	}
	render(w, "home", list)
}

func showPoll(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	p, err := loadPoll(id, voterID(w, r))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	render(w, "poll", p)
}

func panel(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	p, err := loadPoll(id, voterID(w, r))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	render(w, "panel", p)
}

func vote(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	voter := voterID(w, r)
	optID, _ := strconv.ParseInt(r.FormValue("option_id"), 10, 64)
	// Only inserts if the poll is open and the option belongs to it;
	// voting again just moves the voter's choice.
	_, err := db.Exec(`
		INSERT INTO votes(poll_id, option_id, voter)
		SELECT ?, ?, ?
		WHERE EXISTS (SELECT 1 FROM options WHERE id = ? AND poll_id = ?)
		  AND EXISTS (SELECT 1 FROM polls WHERE id = ? AND closed = 0)
		ON CONFLICT(poll_id, voter) DO UPDATE SET option_id = excluded.option_id`,
		id, optID, voter, optID, id, id)
	if err != nil {
		http.Error(w, "db error", 500)
		return
	}
	panel(w, r)
}

func suggest(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	voter := voterID(w, r)
	if label := clip(r.FormValue("label"), maxLabel); label != "" {
		if err := addOption(id, label, true); err != nil {
			http.Error(w, "db error", 500)
			return
		}
	}
	p, err := loadPoll(id, voter)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	p.OOB = true
	render(w, "suggest_response", p)
}
