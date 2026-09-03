package telegram

// Messages are always obtained through ParseExport / ReadExport, so these tests
// describe the adapter's contract to capture — order, text flattening, attachment
// mapping, the UTC timestamp — and nothing about Telegram's format leaks past this
// package.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"expense-reporter/internal/capture"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exportDoc wraps message JSON objects into a full export document.
func exportDoc(messages ...string) string {
	return `{"name":"Gastos","type":"private_group","id":1,"messages":[` + strings.Join(messages, ",") + "]}"
}

// messageJSON builds one "message" entry; extraFields is "" or starts with a comma.
func messageJSON(id int, date string, extraFields string) string {
	return `{"id":` + strconv.Itoa(id) + `,"type":"message","date":"` + date + `","from":"A","from_id":"user1"` + extraFields + "}"
}

// parsed calls ParseExport on the given document and asserts no error.
func parsed(t *testing.T, doc string) []capture.Message {
	t.Helper()
	msgs, err := ParseExport(strings.NewReader(doc))
	require.NoError(t, err)
	return msgs
}

func TestParseExport_KeepsExportOrder(t *testing.T) {
	doc := exportDoc(
		messageJSON(3, "2025-05-03T10:00:00", `,"text":"x; 03/05; 1,00"`),
		messageJSON(1, "2025-05-03T10:00:00", `,"text":"x; 03/05; 1,00"`),
		messageJSON(2, "2025-05-03T10:00:00", `,"text":"x; 03/05; 1,00"`),
	)
	msgs := parsed(t, doc)
	require.Len(t, msgs, 3)
	assert.Equal(t, []int{3, 1, 2}, []int{msgs[0].ID, msgs[1].ID, msgs[2].ID})
}

func TestParseExport_FlattensText(t *testing.T) {
	tests := []struct {
		name     string
		textJSON string
		want     string
	}{
		{"plain string", `"Padaria; 03/05; 12,50"`, "Padaria; 03/05; 12,50"},
		{"empty string", `""`, ""},
		{"runs: strings and a bold object", `["Tela celular; ", {"type":"bold","text":"08/07"}, "; 90,00"]`, "Tela celular; 08/07; 90,00"},
		{"runs: only objects", `[{"type":"link","text":"http://x"}]`, "http://x"},
		{"runs: empty array", `[]`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := exportDoc(messageJSON(1, "2025-05-03T10:00:00", `,"text":`+tt.textJSON))
			msgs := parsed(t, doc)
			require.Len(t, msgs, 1)
			assert.Equal(t, tt.want, msgs[0].Text)
		})
	}
}

func TestParseExport_RejectsUnknownTextShapes(t *testing.T) {
	tests := []struct {
		name     string
		textJSON string
	}{
		{"number", `42`},
		{"object", `{"a":1}`},
		{"array with number", `["a", 5]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := exportDoc(messageJSON(7, "2025-05-03T10:00:00", `,"text":`+tt.textJSON))
			_, err := ParseExport(strings.NewReader(doc))
			require.Error(t, err)
			assert.ErrorContains(t, err, "7")
		})
	}
}

func TestParseExport_MapsAttachments(t *testing.T) {
	tests := []struct {
		name  string
		extra string
		want  capture.Attachment
	}{
		{"photo", `,"text":"","photo":"photos/photo_1.jpg"`, capture.PhotoAttachment},
		{"document", `,"text":"","file":"files/boleto.pdf","mime_type":"application/pdf"`, capture.FileAttachment},
		{"document not exported still counts", `,"text":"","file":"(File not included. Change data exporting settings to download.)"`, capture.FileAttachment},
		{"no media", `,"text":"oi"`, capture.NoAttachment},
		{"file wins over photo when both present", `,"text":"","file":"f.pdf","photo":"p.jpg"`, capture.FileAttachment},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := exportDoc(messageJSON(1, "2025-05-03T10:00:00", tt.extra))
			msgs := parsed(t, doc)
			require.Len(t, msgs, 1)
			assert.Equal(t, tt.want, msgs[0].Attachment)
		})
	}
}

func TestParseExport_SentAtIsTheExportTimestampInUTC(t *testing.T) {
	doc := exportDoc(messageJSON(1, "2025-05-01T22:36:34", `,"text":"x"`))
	msgs := parsed(t, doc)
	require.Len(t, msgs, 1)
	want := time.Date(2025, time.May, 1, 22, 36, 34, 0, time.UTC)
	assert.True(t, want.Equal(msgs[0].SentAt))
	assert.Equal(t, time.UTC, msgs[0].SentAt.Location())
}

func TestParseExport_DropsServiceEntries(t *testing.T) {
	doc := exportDoc(
		`{"id":13,"type":"service","date":"2025-05-01T00:00:00","actor":"A","actor_id":"user1","action":"create_group","title":"Gastos","members":[],"text":"","text_entities":[]}`,
		messageJSON(1, "2025-05-03T10:00:00", `,"text":"x"`),
	)
	msgs := parsed(t, doc)
	require.Len(t, msgs, 1)
	assert.Equal(t, 1, msgs[0].ID)
}

func TestParseExport_Errors(t *testing.T) {
	tests := []struct {
		name         string
		doc          string
		wantContains string
		wantErr      bool
	}{
		{"invalid json", `{"messages": [`, "", true},
		{"unparseable date", exportDoc(messageJSON(9, "2025-13-45T00:00:00", `,"text":"x"`)), "9", true},
		{"no messages key", `{"name":"Gastos"}`, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseExport(strings.NewReader(tt.doc))
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantContains != "" {
					assert.ErrorContains(t, err, tt.wantContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestReadExport(t *testing.T) {
	t.Run("reads valid export", func(t *testing.T) {
		doc := exportDoc(messageJSON(42, "2025-05-03T10:00:00", `,"text":"x"`))
		path := filepath.Join(t.TempDir(), "result.json")
		require.NoError(t, os.WriteFile(path, []byte(doc), 0o644))

		msgs, err := ReadExport(path)
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.Equal(t, 42, msgs[0].ID)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := ReadExport(filepath.Join(t.TempDir(), "missing.json"))
		require.Error(t, err)
	})
}
