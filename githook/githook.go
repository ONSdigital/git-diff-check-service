// Package githook deals with parsing and manipulating githook requests and data
package githook

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

// Common Event Components
type (
	// GithubUser represents a subset of github user data
	GithubUser struct {
		Name      string `json:"name,omitempty"`
		Email     string `json:"email,omitempty"`
		Login     string `json:"login,omitempty"`
		ID        int    `json:"id,omitempty"`
		AvatarURL string `json:"avatar_url,omitempty"`
	}

	// Repository represents a subset of github repository data
	Repository struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
	}

	// Commit represents a subset of github commit data
	Commit struct {
		URL   string `json:"url"`
		SHA   string `json:"sha"`
		Files []struct {
			Filename  string `json:"filename"`
			Additions int    `json:"additions"`
			Deletions int    `json:"deletions"`
			Changes   int    `json:"changes"`
			Status    string `json:"status"`
			Patch     string `json:"patch"`
		} `json:"files"`
		Commit struct {
			Author    GithubUser `json:"author"`
			Committer GithubUser `json:"committer"`
		} `json:"commit"`
		Message string `json:"message"`
	}
)

type (
	// PushEvent is a subset of data returned from a "push" event webhook
	PushEvent struct {
		Ref     string `json:"id"`
		Commits []struct {
			ID string `json:"id"`
		} `json:"commits"`
		Repository Repository `json:"repository"`
		Pusher     GithubUser `json:"pusher"`
	}

	// PingEvent is a subset of data returned from a "ping" event webhook
	PingEvent struct {
		Repository Repository `json:"repository"`
	}
)

const (
	signaturePrefix = "sha1="
)

var (
	// ErrUnsupportedEvent is when service recieves an event type other than
	// 'ping' or 'push'
	ErrUnsupportedEvent = errors.New("only supports 'push' and 'ping' events")

	// ErrMissingEvent is when the x-github-event header is either missing or empty
	ErrMissingEvent = errors.New("missing or empty x-github-event header")

	// ErrBadSignature is when the signature check fails against an incoming
	// event payload
	ErrBadSignature = errors.New("bad signature")

	// ErrUnsupportedContentType is when a parsed request is not application/json
	ErrUnsupportedContentType = errors.New("unsupported content-type, must be application/json")

	// ErrBodyUnreadable is when payload content cannot be read from a request
	ErrBodyUnreadable = errors.New("unable to read request content")
)

// Parse attempts to parse webhook data from a http.Request. If successful, returns
// the event type and an event struct
func Parse(r *http.Request, secret []byte) (interface{}, error) {

	if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
		log.Println("GOT", contentType)
		return nil, ErrUnsupportedContentType
	}

	var signature string
	if signature = r.Header.Get("x-hub-signature"); !strings.HasPrefix(signature, signaturePrefix) {
		return nil, errors.New("missing x-hub-signature")
	}
	signature = strings.TrimLeft(signature, signaturePrefix)

	var eventType string
	if eventType = r.Header.Get("x-github-event"); len(eventType) == 0 {
		return nil, ErrMissingEvent
	}

	var payload []byte
	var err error
	if payload, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, ErrBodyUnreadable
	}

	// ConstantTimeCompare to mitigate possible timing attacks
	if subtle.ConstantTimeCompare(SignPayload(payload, secret), []byte(signature)) != 1 {
		return nil, ErrBadSignature
	}

	switch eventType {
	case "ping":
		return unmarshalEvent(payload, &PingEvent{})
	case "push":
		return unmarshalEvent(payload, &PushEvent{})
	default:
		return nil, ErrUnsupportedEvent
	}
}

func unmarshalEvent(payload []byte, event interface{}) (interface{}, error) {
	err := json.Unmarshal(payload, &event)
	return event, err
}

// SignPayload uses the given secret to sign the payload data using the same
// method as github
func SignPayload(payload, secret []byte) []byte {
	computed := hmac.New(sha1.New, secret)
	computed.Write(payload)
	// Return the hex encoded representation of the signature so it can be
	// plugged directly into a header (and compared with an existing header)
	return []byte(fmt.Sprintf("%x", computed.Sum(nil)))
}
