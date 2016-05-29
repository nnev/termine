package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"text/template"
	"time"
)

var cmdAnnounce = &Command{
	UsageLine: "announce",
	Short:     "Kündigt nächsten Stammtisch oder nächste c¼h an",
	Long: `Announced den nächsten Stammtisch oder die nächste c¼h,
je nachdem, was am nächsten Donnerstag ist.`,
	Flag:         flag.NewFlagSet("announce", flag.ExitOnError),
	NeedsDB:      true,
	RegenWebsite: false,
}

func init() {
	cmdAnnounce.Run = RunAnnounce
}

func isStammtisch(date time.Time) (stammt bool, err error) {
	err = db.QueryRow("SELECT stammtisch FROM termine WHERE date = $1", date).Scan(&stammt)
	return
}

func announceStammtisch(date time.Time) {
	loc, err := getLocation(date)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Kann Location nicht auslesen:", err)
		return
	}

	maildraft := `Liebe Treffler,

am kommenden Donnerstag ist wieder Stammtisch. Diesmal sind wir bei {{.Location}}.

Damit wir passend reservieren können, tragt bitte bis Dienstag Abend,
18:00 Uhr unter [0] ein, ob ihr kommt oder nicht.


[0] https://www.noname-ev.de/yarpnarp.html
	`

	mailtmpl := template.Must(template.New("maildraft").Parse(maildraft))
	mailbuf := new(bytes.Buffer)
	type data struct {
		Location string
	}
	if err = mailtmpl.Execute(mailbuf, data{loc}); err != nil {
		fmt.Fprintln(os.Stderr, "Fehler beim Füllen des Templates:", err)
		return
	}
	mail := mailbuf.Bytes()

	sendAnnouncement("Bitte für Stammtisch eintragen", mail)
}

func announceC14(date time.Time) {
	var data struct {
		Topic,
		Abstract,
		Speaker string
	}

	if err := db.QueryRow("SELECT topic FROM vortraege WHERE date = $1", date).Scan(&data.Topic); err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Es gibt nächsten Donnerstag noch keine c¼h. :(")
			return
		}

		fmt.Fprintln(os.Stderr, "Kann topic nicht auslesen:", err)
		return
	}

	if err := db.QueryRow("SELECT abstract FROM vortraege WHERE date = $1", date).Scan(&data.Abstract); err != nil {
		fmt.Fprintln(os.Stderr, "Kann abstract nicht auslesen:", err)
		return
	}

	if err := db.QueryRow("SELECT speaker FROM vortraege WHERE date = $1", date).Scan(&data.Speaker); err != nil {
		fmt.Fprintln(os.Stderr, "Kann speaker nicht auslesen:", err)
		return
	}

	maildraft := `Liebe Treffler,

am kommenden Donnerstag wird {{.Speaker}} eine c¼h zum Thema

    {{.Topic}}

halten.

Kommet zahlreich!


Wer mehr Informationen möchte:

{{.Abstract}}
	`

	mailtmpl := template.Must(template.New("maildraft").Parse(maildraft))
	mailbuf := new(bytes.Buffer)
	if err := mailtmpl.Execute(mailbuf, data); err != nil {
		fmt.Fprintln(os.Stderr, "Fehler beim Füllen des Templates:", err)
		return
	}
	mail := mailbuf.Bytes()
	sendAnnouncement(data.Topic, mail)
}

func sendAnnouncement(subject string, msg []byte) error {
	fromheader := "From: frank@noname-ev.de"
	toheader := "To: ccchd@ccchd.de"
	subjectheader := "Subject: " + subject
	fullmail := []byte(fromheader + "\n" + subjectheader + "\n" + toheader + "\n\n")
	fullmail = append(fullmail, msg...)

	cmd := exec.Command("/usr/sbin/sendmail", "-t")

	cmd.Stdin = bytes.NewReader(fullmail)

	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Fehler beim Senden: ", err)
	}

	return nil
}

func RunAnnounce() {
	nextThursday := getNextThursdays(1)[0]

	isStm, err := isStammtisch(nextThursday)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Kann stammtischiness nicht auslesen:", err)
		return
	}

	if isStm {
		announceStammtisch(nextThursday)
	} else {
		announceC14(nextThursday)
	}
}
