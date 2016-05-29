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
	Short:     "Announced nächsten Stammtisch oder c1/4",
	Long: `Announced den nächsten Stammtisch oder die nächste c1/4,
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
	err = mailtmpl.Execute(mailbuf, data{loc})
	mail := mailbuf.String()

	sendAnnouncement("Bitte für Stammtisch eintragen", mail)

}

func announceC14(date time.Time) {

	type data struct {
		Topic, Abstract, Speaker string
	}

	d := new(data)

	err := db.QueryRow("SELECT topic FROM vortraege WHERE date = $1", date).Scan(&d.Topic)
	if err != nil {

		if err == sql.ErrNoRows {
			fmt.Println("Es gibt nächsten Donnerstag noch keine c1/4. :(")
			return
		}

		fmt.Fprintln(os.Stderr, "Kann topic nicht auslesen:", err)
		return
	}

	err = db.QueryRow("SELECT abstract FROM vortraege WHERE date = $1", date).Scan(&d.Abstract)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Kann abstract nicht auslesen:", err)
		return
	}

	err = db.QueryRow("SELECT speaker FROM vortraege WHERE date = $1", date).Scan(&d.Speaker)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Kann speaker nicht auslesen:", err)
		return
	}

	maildraft := `Liebe Treffler,

am kommenden Donnerstag wird es um das Thema

    {{.Topic}}

gehen. Es spricht {{.Speaker}} zu uns.


Kommet zahlreich!


Wer mehr Informationen möchte:

{{.Abstract}}
	`

	mailtmpl := template.Must(template.New("maildraft").Parse(maildraft))
	mailbuf := new(bytes.Buffer)
	err = mailtmpl.Execute(mailbuf, d)
	mail := mailbuf.String()
	sendAnnouncement(d.Topic, mail)
}

func sendAnnouncement(subject, msg string) error {
	fromheader := "From: termine@noname-ev.de"
	toheader := "To: cherti@letopolis.de"
	subjectheader := "Subject: " + subject
	fullmail := fromheader + "\n" + subjectheader + "\n" + toheader + "\n\n" + msg

	fmt.Println(":: creating command")
	cmd := exec.Command("/usr/sbin/sendmail", "-t")

	fmt.Println(":: setting up stdin")
	indump, err := cmd.StdinPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Fehler beim Senden: ", err)
	}

	fmt.Println(":: write to stdin")
	_, err = indump.Write([]byte(fullmail))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Fehler beim Senden: ", err)
	}
	indump.Close()

	err = cmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Fehler beim Senden: ", err)
	}

	return nil
}

func RunAnnounce() {
	// get next donnerstag, bool stammtisch is true oder false
	nextThursday := getNextThursdays(3)[2]

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