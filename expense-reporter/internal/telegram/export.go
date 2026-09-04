// Package telegram is a SOURCE ADAPTER: it reads what Telegram Desktop's "Export
// chat history" writes (result.json) and hands the messages to internal/capture as
// capture.Messages. It knows the export's JSON shape and nothing about expenses —
// no parsing, no repair, no classification happens here. That is the split the
// T-75 plan rests on: a second adapter (the bot) will produce the same Messages,
// and capture will read them identically.
package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"expense-reporter/internal/capture"
)

// exportFile is the top level of result.json. Only what the adapter reads is
// declared; the export carries much more (participants, reactions, edit times).
type exportFile struct {
	Name     string          `json:"name"`
	Messages []exportMessage `json:"messages"`
}

// exportMessage is one entry of the messages array. Text is kept raw because the
// export writes it EITHER as a plain string OR as an array of runs — strings mixed
// with {"type": "bold", "text": "..."} objects — whenever the message had formatting.
type exportMessage struct {
	ID    int             `json:"id"`
	Type  string          `json:"type"`
	Date  string          `json:"date"`
	Text  json.RawMessage `json:"text"`
	File  string          `json:"file"`
	Photo string          `json:"photo"`
}

// dateLayout is the export's local-time timestamp, no zone ("2025-05-01T22:36:34").
// It is read in UTC on purpose: the calendar DAY is what capture uses as year
// evidence, and it must be the sender's day, not the day after a zone shift.
const dateLayout = "2006-01-02T15:04:05"

// ReadExport reads a result.json from disk and returns its messages in export order.
func ReadExport(path string) ([]capture.Message, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening telegram export: %w", err)
	}
	defer f.Close()
	msgs, err := ParseExport(f)
	if err != nil {
		return nil, fmt.Errorf("reading telegram export %s: %w", path, err)
	}
	return msgs, nil
}

// ParseExport decodes an export and converts every entry of type "message" into a
// capture.Message, in order. Entries of any other type (Telegram's "service"
// entries: group created, member joined) are dropped — they are not something a
// human typed. An entry whose date does not parse, or whose text is neither a
// string nor an array of runs, is an error naming the message id: an export that
// cannot be read completely must not be read partially.
func ParseExport(r io.Reader) ([]capture.Message, error) {
	var ef exportFile
	if err := json.NewDecoder(r).Decode(&ef); err != nil {
		return nil, fmt.Errorf("decoding export: %w", err)
	}

	msgs := make([]capture.Message, 0)
	for _, m := range ef.Messages {
		if m.Type != "message" {
			continue
		}
		cmsg, err := toMessage(m)
		if err != nil {
			return nil, fmt.Errorf("message %d: %w", m.ID, err)
		}
		msgs = append(msgs, cmsg)
	}
	return msgs, nil
}

// toMessage converts one "message" entry.
func toMessage(m exportMessage) (capture.Message, error) {
	text, err := flattenText(m.Text)
	if err != nil {
		return capture.Message{}, fmt.Errorf("flattening text: %w", err)
	}
	sentAt, err := sentAt(m.Date)
	if err != nil {
		return capture.Message{}, fmt.Errorf("parsing date: %w", err)
	}
	attachment := attachmentOf(m)
	return capture.Message{
		ID:         m.ID,
		SentAt:     sentAt,
		Text:       text,
		Attachment: attachment,
	}, nil
}

// flattenText returns the plain text of a message: the string itself, or the
// concatenation of an array of runs where each run is either a string or an
// object carrying a "text" field. Any other JSON shape is an error.
func flattenText(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}

	var runs []json.RawMessage
	if err := json.Unmarshal(raw, &runs); err != nil {
		return "", fmt.Errorf("text is neither a string nor an array of runs: %w", err)
	}

	return joinRuns(runs)
}

// joinRuns concatenates the text from each run in the slice.
func joinRuns(runs []json.RawMessage) (string, error) {
	var sb strings.Builder
	for i, r := range runs {
		text, err := runText(r)
		if err != nil {
			return "", fmt.Errorf("run %d: %w", i, err)
		}
		sb.WriteString(text)
	}
	return sb.String(), nil
}

// runText extracts the text from a single run.
func runText(raw json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}

	var obj struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", fmt.Errorf("run is neither a string nor an object with text field")
	}
	return obj.Text, nil
}

// attachmentOf maps the export's media fields: a non-empty "file" is a
// FileAttachment (documents, PDFs — the value may be a placeholder like
// "(File not included. ...)" when media was not exported, which still counts),
// a non-empty "photo" is a PhotoAttachment, neither is NoAttachment.
func attachmentOf(m exportMessage) capture.Attachment {
	if m.File != "" {
		return capture.FileAttachment
	}
	if m.Photo != "" {
		return capture.PhotoAttachment
	}
	return capture.NoAttachment
}

// sentAt parses the export timestamp with dateLayout in UTC.
func sentAt(date string) (time.Time, error) {
	t, err := time.Parse(dateLayout, date)
	if err != nil {
		return time.Time{}, fmt.Errorf("date %q: %w", date, err)
	}
	return t, nil
}
