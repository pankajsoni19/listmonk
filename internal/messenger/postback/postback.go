package postback

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/knadh/listmonk/internal/balancer"
	"github.com/knadh/listmonk/models"
)

// postback is the payload that's posted as JSON to the HTTP Postback server.
//
//easyjson:json
type postback struct {
	FromEmail   string       `json:"from_email"`
	Subject     string       `json:"subject"`
	ContentType string       `json:"content_type"`
	Body        string       `json:"body"`
	Recipients  []recipient  `json:"recipients"`
	Campaign    *campaign    `json:"campaign"`
	Attachments []attachment `json:"attachments"`
}

type campaign struct {
	FromEmail string         `json:"from_email"`
	UUID      string         `json:"uuid"`
	Name      string         `json:"name"`
	Headers   models.Headers `json:"headers"`
	Tags      []string       `json:"tags"`
}

type recipient struct {
	UUID    string      `json:"uuid"`
	Email   string      `json:"email"`
	Name    string      `json:"name"`
	Attribs models.JSON `json:"attribs"`
	Status  string      `json:"status"`
}

type attachment struct {
	Name    string               `json:"name"`
	Header  textproto.MIMEHeader `json:"header"`
	Content []byte               `json:"content"`
}

// Options represents HTTP Postback server options.
type Options struct {
	UUID     string        `json:"uuid"`
	Name     string        `json:"name"`
	Username string        `json:"username"`
	Password string        `json:"password"`
	RootURL  string        `json:"root_url"`
	MaxConns int           `json:"max_conns"`
	Retries  int           `json:"retries"`
	Timeout  time.Duration `json:"timeout"`
	WFrom    string        `json:"wfrom"`
}

// Postback represents an HTTP Message server.
type Postback struct {
	authStr  string
	o        Options
	c        *http.Client
	balancer *balancer.Balance
}

func (o Options) makeBalancer() *balancer.Balance {
	// copied from email
	parts := strings.Split(o.WFrom, ",")

	var lastKey string
	choiceVal := make(map[string]int)

	for idx, part := range parts {
		spart := strings.TrimSpace(part)

		if len(parts) == idx+1 && len(spart) == 0 {
			break
		}

		if len(lastKey) > 0 {
			if v, e := strconv.Atoi(spart); e == nil {
				choiceVal[lastKey] = v
				lastKey = ""
				continue
			}
		}

		choiceVal[spart] = 1
		lastKey = spart
	}

	balancer := balancer.NewBalance()

	for k, v := range choiceVal {
		balancer.Add(k, v)
	}

	return balancer
}

// New returns a new instance of the HTTP Postback messenger.
func New(o Options) (*Postback, error) {
	authStr := ""
	if o.Username != "" && o.Password != "" {
		authStr = fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString(
			[]byte(o.Username+":"+o.Password)))
	}

	return &Postback{
		authStr: authStr,
		o:       o,
		c: &http.Client{
			Timeout: o.Timeout,
			Transport: &http.Transport{
				MaxIdleConnsPerHost:   o.MaxConns,
				MaxConnsPerHost:       o.MaxConns,
				ResponseHeaderTimeout: o.Timeout,
				IdleConnTimeout:       o.Timeout,
			},
		},
		balancer: o.makeBalancer(),
	}, nil
}

// Name returns the messenger's name.
func (p *Postback) Name() string {
	return p.o.Name
}

func (p *Postback) UUID() string {
	return p.o.UUID
}

func (p *Postback) IsDefault() bool {
	return false
}

// Push pushes a message to the server.
func (p *Postback) Push(m models.Message) error {
	from := p.balancer.Get()

	pb := postback{
		FromEmail:   from,
		Subject:     m.Subject,
		ContentType: m.ContentType,
		Body:        string(m.Body),
		Recipients: []recipient{{
			UUID:    m.Subscriber.UUID,
			Email:   m.Subscriber.Email,
			Name:    m.Subscriber.Name,
			Status:  m.Subscriber.Status,
			Attribs: m.Subscriber.Attribs,
		}},
	}

	if m.Campaign != nil {
		pb.Campaign = &campaign{
			FromEmail: from,
			UUID:      m.Campaign.UUID,
			Name:      m.Campaign.Name,
			Headers:   m.Campaign.Headers,
			Tags:      m.Campaign.Tags,
		}
	}

	if len(m.Attachments) > 0 {
		files := make([]attachment, 0, len(m.Attachments))
		for _, f := range m.Attachments {
			a := attachment{
				Name:    f.Name,
				Header:  f.Header,
				Content: make([]byte, len(f.Content)),
			}
			copy(a.Content, f.Content)
			files = append(files, a)
		}
		pb.Attachments = files
	}

	b, err := json.Marshal(pb)
	if err != nil {
		return err
	}

	return p.exec(http.MethodPost, p.o.RootURL, b, nil)
}

// Flush flushes the message queue to the server.
func (p *Postback) Flush() error {
	return nil
}

// Close closes idle HTTP connections.
func (p *Postback) Close() error {
	p.c.CloseIdleConnections()
	return nil
}

func (p *Postback) exec(method, rURL string, reqBody []byte, headers http.Header) error {
	var (
		err      error
		postBody io.Reader
	)

	// Encode POST / PUT params.
	if method == http.MethodPost || method == http.MethodPut {
		postBody = bytes.NewReader(reqBody)
	}

	req, err := http.NewRequest(method, rURL, postBody)
	if err != nil {
		return err
	}

	if headers != nil {
		req.Header = headers
	} else {
		req.Header = http.Header{}
	}
	req.Header.Set("User-Agent", "listmonk")

	// Optional BasicAuth.
	if p.authStr != "" {
		req.Header.Set("Authorization", p.authStr)
	}

	// If a content-type isn't set, set the default one.
	if req.Header.Get("Content-Type") == "" {
		if method == http.MethodPost || method == http.MethodPut {
			req.Header.Add("Content-Type", "application/json")
		}
	}

	// If the request method is GET or DELETE, add the params as QueryString.
	if method == http.MethodGet || method == http.MethodDelete {
		req.URL.RawQuery = string(reqBody)
	}

	r, err := p.c.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		// Drain and close the body to let the Transport reuse the connection
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
	}()

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("non-OK response from Postback server: %d", r.StatusCode)
	}

	return nil
}
