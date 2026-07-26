package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/psanford/wormhole-william/wormhole"
)

const (
	// currentShareVersion is the version field written into every payload.
	currentShareVersion = 1
	// maxPayloadBytes is the maximum wormhole payload size we will accept.
	// This prevents a malicious or buggy peer from allocating unbounded memory.
	maxPayloadBytes = 10 * 1024 * 1024 // 10 MB
	// maxSenderLabelLen is the max bytes we accept for a sender label string.
	maxSenderLabelLen = 32
)

// SharePayload is the JSON envelope sent through the wormhole.
// It carries the tab title, body, and an optional human label for the sender.
type SharePayload struct {
	TabTitle    string `json:"tab_title"`
	Body        string `json:"body"`
	SenderLabel string `json:"sender_label,omitempty"` // e.g. "Alice" — purely informational
	Version     int    `json:"version"`
}

// ShareResult is returned to callers after a successful receive.
type ShareResult struct {
	TabTitle    string
	Body        string
	SenderLabel string // empty string if the sender did not provide one
}

// ShareSend serialises the given tab and opens a Magic Wormhole.
// senderLabel is an optional human-readable name for the sender (e.g. "Alice").
// It is embedded in the encrypted payload and shown to the receiver as context.
// Pass an empty string to omit it.
//
// It returns the human-friendly code (e.g. "7-crossover-alpha") immediately,
// then blocks on the returned wait func until the peer has received the data.
// The caller should pass a cancellable ctx to abort waiting.
func ShareSend(ctx context.Context, tab Tab, senderLabel string) (code string, wait func() error, err error) {
	// Sanitise sender label: trim, strip control chars, max length.
	senderLabel = sanitiseSenderLabel(senderLabel)

	// Sanitise body — strip NUL bytes before sending.
	body := strings.ReplaceAll(tab.Body, "\x00", "")

	// Reject bodies that exceed the max payload size.
	if len(body) > maxPayloadBytes {
		return "", nil, fmt.Errorf("share: tab body too large (%d bytes > %d byte limit)", len(body), maxPayloadBytes)
	}

	payload := SharePayload{
		TabTitle:    SanitiseTitle(tab.Title),
		Body:        body,
		SenderLabel: senderLabel,
		Version:     currentShareVersion,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", nil, fmt.Errorf("share: marshal payload: %w", err)
	}

	var c wormhole.Client
	code, statusCh, err := c.SendText(ctx, string(data))
	if err != nil {
		return "", nil, fmt.Errorf("share: open wormhole: %w", err)
	}

	wait = func() error {
		s, ok := <-statusCh
		if !ok {
			return fmt.Errorf("share: connection closed before transfer")
		}
		if s.Error != nil {
			return fmt.Errorf("share: transfer error: %w", s.Error)
		}
		return nil
	}

	return code, wait, nil
}

// ShareReceive connects to an existing wormhole using the code typed by the user.
// It blocks until the text is fully received, then returns the decoded tab content.
func ShareReceive(ctx context.Context, code string) (ShareResult, error) {
	// Sanitise the code input before passing to wormhole library.
	code = strings.TrimSpace(code)
	if code == "" {
		return ShareResult{}, fmt.Errorf("share: empty code")
	}

	var c wormhole.Client

	msg, err := c.Receive(ctx, code)
	if err != nil {
		return ShareResult{}, fmt.Errorf("share: receive handshake: %w", err)
	}

	if msg.Type != wormhole.TransferText {
		_ = msg.Reject()
		return ShareResult{}, fmt.Errorf("share: peer sent a file, not text — wrong tool?")
	}

	// Limit the reader to maxPayloadBytes to prevent memory exhaustion.
	limited := io.LimitReader(msg, maxPayloadBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return ShareResult{}, fmt.Errorf("share: read message: %w", err)
	}

	// Reject payloads that hit or exceed the size limit.
	if len(raw) > maxPayloadBytes {
		return ShareResult{}, fmt.Errorf("share: received payload too large (> %d MB limit)", maxPayloadBytes/1024/1024)
	}

	// Strip NUL bytes from raw payload before parsing.
	raw = []byte(strings.ReplaceAll(string(raw), "\x00", ""))

	// Validate that payload is valid UTF-8.
	if !utf8.Valid(raw) {
		return ShareResult{}, fmt.Errorf("share: received payload is not valid UTF-8")
	}

	var payload SharePayload
	if jsonErr := json.Unmarshal(raw, &payload); jsonErr != nil {
		// Graceful fallback: treat the raw text as the tab body if it's not
		// our JSON envelope (e.g. shared from the reference Python client).
		return ShareResult{
			TabTitle: "shared note",
			Body:     string(raw),
		}, nil
	}

	// Version checking: warn on forward-incompatible versions but still accept.
	if payload.Version < 0 || payload.Version > 100 {
		return ShareResult{}, fmt.Errorf("share: received payload has invalid version %d", payload.Version)
	}

	// Sanitise received fields.
	title := SanitiseTitle(payload.TabTitle)
	if title == "" {
		title = "shared note"
	}
	senderLabel := sanitiseSenderLabel(payload.SenderLabel)
	body := strings.ReplaceAll(payload.Body, "\x00", "")

	// Validate body UTF-8.
	if !utf8.ValidString(body) {
		body = strings.ToValidUTF8(body, "")
	}

	// Prefix the tab title with the sender's label so the receiver immediately
	// knows where the content came from, e.g. "From Alice · my-notes"
	if senderLabel != "" {
		title = "From " + senderLabel + " · " + title
	}
	return ShareResult{
		TabTitle:    title,
		Body:        body,
		SenderLabel: senderLabel,
	}, nil
}

// sanitiseSenderLabel trims, strips control characters, and truncates a sender label.
func sanitiseSenderLabel(label string) string {
	label = strings.TrimSpace(label)
	var b strings.Builder
	for _, r := range label {
		if r >= 0x20 && r != 0x7F {
			b.WriteRune(r)
		}
	}
	label = b.String()
	if len(label) > maxSenderLabelLen {
		label = label[:maxSenderLabelLen]
	}
	return label
}
