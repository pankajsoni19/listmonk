package main

import (
	"bytes"
	"net/textproto"
	"regexp"
	"strings"

	"github.com/knadh/listmonk/models"
)

const (
	notifTplImport       = "import-status"
	notifTplCampaign     = "campaign-status"
	notifSubscriberOptin = "subscriber-optin"
	notifSubscriberData  = "subscriber-data"
)

var (
	reTitle = regexp.MustCompile(`(?s)<title\s*data-i18n\s*>(.+?)</title>`)
)

// sendNotification sends out an e-mail notification to admins.
func (app *App) sendNotification(toEmails []string, subject, tplName string, data interface{}, headers textproto.MIMEHeader) error {
	if len(toEmails) == 0 {
		return nil
	}

	var buf bytes.Buffer
	if err := app.notifTpls.tpls.ExecuteTemplate(&buf, tplName, data); err != nil {
		app.log.Printf("error compiling notification template '%s': %v", tplName, err)
		return err
	}
	body := buf.Bytes()

	subject, body = getTplSubject(subject, body)

	m := models.Message{}
	m.ContentType = app.notifTpls.contentType
	m.To = toEmails
	m.Subject = subject
	m.Body = body
	m.From = app.defaultMessenger.From()
	m.Messenger = app.defaultMessenger.UUID()
	m.Headers = headers

	if err := app.manager.PushMessage(m); err != nil {
		app.log.Printf("error sending admin notification (%s): %v", subject, err)
		return err
	}
	return nil
}

// getTplSubject extrcts any custom i18n subject rendered in the given rendered
// template body. If it's not found, the incoming subject and body are returned.
func getTplSubject(subject string, body []byte) (string, []byte) {
	m := reTitle.FindSubmatch(body)
	if len(m) != 2 {
		return subject, body
	}

	return strings.TrimSpace(string(m[1])), reTitle.ReplaceAll(body, []byte(""))
}
