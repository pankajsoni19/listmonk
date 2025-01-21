package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"net/textproto"
	"strings"

	"github.com/knadh/listmonk/models"
	"github.com/knadh/smtppool"
)

const (
	hdrReturnPath = "Return-Path"
	hdrBcc        = "Bcc"
	hdrCc         = "Cc"
)

// Server represents an SMTP server's credentials.
type Server struct {
	From          string            `json:"from"`
	UUID          string            `json:"uuid"`
	Name          string            `json:"name"`
	Username      string            `json:"username"`
	Password      string            `json:"password"`
	AuthProtocol  string            `json:"auth_protocol"`
	TLSType       string            `json:"tls_type"`
	TLSSkipVerify bool              `json:"tls_skip_verify"`
	EmailHeaders  map[string]string `json:"email_headers"`
	Default       bool              `json:"default"`

	// Rest of the options are embedded directly from the smtppool lib.
	// The JSON tag is for config unmarshal to work.
	smtppool.Opt `json:",squash"`

	pool *smtppool.Pool
}

// Emailer is the SMTP e-mail messenger.
type Emailer struct {
	server *Server
}

// New returns an SMTP e-mail Messenger backend with the given SMTP servers.
func New(s Server) (*Emailer, error) {
	switch s.AuthProtocol {
	case "cram":
		s.Opt.Auth = smtp.CRAMMD5Auth(s.Username, s.Password)
	case "plain":
		s.Opt.Auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
	case "login":
		s.Opt.Auth = &smtppool.LoginAuth{Username: s.Username, Password: s.Password}
	case "", "none":
	default:
		return nil, fmt.Errorf("unknown SMTP auth type '%s'", s.AuthProtocol)
	}

	// TLS config.
	if s.TLSType != "none" {
		s.TLSConfig = &tls.Config{}
		if s.TLSSkipVerify {
			s.TLSConfig.InsecureSkipVerify = s.TLSSkipVerify
		} else {
			s.TLSConfig.ServerName = s.Host
		}

		// SSL/TLS, not STARTTLS.
		if s.TLSType == "TLS" {
			s.Opt.SSL = true
		}
	}

	pool, err := smtppool.New(s.Opt)
	if err != nil {
		return nil, err
	}

	s.pool = pool

	e := &Emailer{
		server: &s,
	}

	return e, nil
}

// Name returns the Server's name.
func (e *Emailer) Name() string {
	return e.server.Name
}

func (e *Emailer) UUID() string {
	return e.server.UUID
}

func (e *Emailer) IsDefault() bool {
	return e.server.Default
}

func (e *Emailer) From() string {
	return e.server.From
}

// Push pushes a message to the server.
func (e *Emailer) Push(m models.Message) error {
	// If there are more than one SMTP servers, send to a random
	// one from the list.
	var srv = e.server

	// Are there attachments?
	var files []smtppool.Attachment
	if m.Attachments != nil {
		files = make([]smtppool.Attachment, 0, len(m.Attachments))
		for _, f := range m.Attachments {
			a := smtppool.Attachment{
				Filename: f.Name,
				Header:   f.Header,
				Content:  make([]byte, len(f.Content)),
			}
			copy(a.Content, f.Content)
			files = append(files, a)
		}
	}

	em := smtppool.Email{
		From:        m.From,
		To:          m.To,
		Subject:     m.Subject,
		Attachments: files,
	}

	em.Headers = textproto.MIMEHeader{}

	// Attach SMTP level headers.
	for k, v := range srv.EmailHeaders {
		em.Headers.Set(k, v)
	}

	// Attach e-mail level headers.
	for k, v := range m.Headers {
		em.Headers.Set(k, v[0])
	}

	// If the `Return-Path` header is set, it should be set as the
	// the SMTP envelope sender (via the Sender field of the email struct).
	if sender := em.Headers.Get(hdrReturnPath); sender != "" {
		em.Sender = sender
		em.Headers.Del(hdrReturnPath)
	}

	// If the `Bcc` header is set, it should be set on the Envelope
	if bcc := em.Headers.Get(hdrBcc); bcc != "" {
		for _, part := range strings.Split(bcc, ",") {
			em.Bcc = append(em.Bcc, strings.TrimSpace(part))
		}
		em.Headers.Del(hdrBcc)
	}

	// If the `Cc` header is set, it should be set on the Envelope
	if cc := em.Headers.Get(hdrCc); cc != "" {
		for _, part := range strings.Split(cc, ",") {
			em.Cc = append(em.Cc, strings.TrimSpace(part))
		}
		em.Headers.Del(hdrCc)
	}

	switch m.ContentType {
	case "plain":
		em.Text = []byte(m.Body)
	default:
		em.HTML = m.Body
		if len(m.AltBody) > 0 {
			em.Text = m.AltBody
		}
	}

	return srv.pool.Send(em)
}

// Flush flushes the message queue to the server.
func (e *Emailer) Flush() error {
	return nil
}

// Close closes the SMTP pools.
func (e *Emailer) Close() error {
	e.server.pool.Close()
	return nil
}
