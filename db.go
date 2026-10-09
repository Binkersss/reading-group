package main

import (
	"database/sql"
	"strconv"
)

const schema = `
CREATE TABLE IF NOT EXISTS polls (
	id INTEGER PRIMARY KEY,
	question TEXT NOT NULL,
	closed INTEGER NOT NULL DEFAULT 0,
	allow_suggest INTEGER NOT NULL DEFAULT 1,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS options (
	id INTEGER PRIMARY KEY,
	poll_id INTEGER NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
	label TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS votes (
	poll_id INTEGER NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
	option_id INTEGER NOT NULL REFERENCES options(id) ON DELETE CASCADE,
	voter TEXT NOT NULL,
	PRIMARY KEY (poll_id, voter)
);
CREATE INDEX IF NOT EXISTS votes_option ON votes(option_id);
CREATE INDEX IF NOT EXISTS options_poll ON options(poll_id);
`
const (
	maxOptions   = 12
	maxQuestions = 128
	maxLabel     = 120
)

var db *sql.DB

type Opt struct {
	ID    int64
	Label string
	Votes int
	Pct   int
	Mine  bool
}

type Poll struct {
	ID           int64
	Question     string
	Closed       bool
	AllowSuggest bool
	ReadOnly     bool   // admin view: results only, no vote buttons
	PanelURL     string // where the results panel refreshes from
	OOB          bool   // render panel as an out-of-band swap
	Total        int
	Options      []Opt
}

type PollRow struct {
	ID       int64
	Question string
	Closed   bool
	Votes    int
}

func loadPoll(id int64, voter string) (*Poll, error) {
	p := &Poll{ID: id, PanelURL: "/polls/" + strconv.FormatInt(id, 10) + "/panel"}
	var closed, allow int
	err := db.QueryRow(`SELECT question, closed, allow_suggest FROM polls WHERE id = ?`, id).
		Scan(&p.Question, &closed, &allow)
	if err != nil {
		return nil, err
	}
	p.Closed = closed != 0
	p.AllowSuggest = allow != 0

	var mine int64
	_ = db.QueryRow(`SELECT option_id FROM votes WHERE poll_id = ? AND voter = ?`, id, voter).Scan(&mine)

	rows, err := db.Query(`
		SELECT o.id, o.label, COUNT(v.voter)
		FROM options o LEFT JOIN votes v ON v.option_id = o.id
		WHERE o.poll_id = ? GROUP BY o.id ORDER BY o.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var o Opt
		if err := rows.Scan(&o.ID, &o.Label, &o.Votes); err != nil {
			return nil, err
		}
		o.Mine = o.ID == mine
		p.Total += o.Votes
		p.Options = append(p.Options, o)
	}
	for i := range p.Options {
		if p.Total > 0 {
			p.Options[i].Pct = p.Options[i].Votes * 100 / p.Total
		}
	}
	return p, rows.Err()
}

func loadAdminPoll(id int64) (*Poll, error) {
	p, err := loadPoll(id, "")
	if err != nil {
		return nil, err
	}
	p.ReadOnly = true
	p.PanelURL = "/admin/polls/" + strconv.FormatInt(id, 10) + "/panel"
	return p, nil
}

func addOption(pollID int64, label string, public bool) error {
	cond := `EXISTS (SELECT 1 FROM polls WHERE id = ?)`
	if public {
		cond = `EXISTS (SELECT 1 FROM polls WHERE id = ? AND closed = 0 AND allow_suggest = 1)`
	}
	_, err := db.Exec(`
		INSERT INTO options(poll_id, label)
		SELECT ?, ?
		WHERE `+cond+`
		  AND (SELECT COUNT(*) FROM options WHERE poll_id = ?) < ?
		  AND NOT EXISTS (SELECT 1 FROM options WHERE poll_id = ? AND lower(label) = lower(?))`,
		pollID, label, pollID, pollID, maxOptions, pollID, label)
	return err
}

func listPolls() ([]PollRow, error) {
	rows, err := db.Query(`
		SELECT p.id, p.question, p.closed,
		       (SELECT COUNT(*) FROM votes WHERE poll_id = p.id)
		FROM polls p ORDER BY p.closed, p.id DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []PollRow
	for rows.Next() {
		var pr PollRow
		var c int
		if err := rows.Scan(&pr.ID, &pr.Question, &c, &pr.Votes); err != nil {
			return nil, err
		}
		pr.Closed = c != 0
		list = append(list, pr)
	}
	return list, rows.Err()
}
