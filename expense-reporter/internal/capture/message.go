// Package capture is the source-agnostic half of expense capture. It takes the
// messages a human typed into the chat group — from a Telegram export today, from a
// bot tomorrow — and decides, for each one, whether it is an expense the batch
// pipeline can take as-is, an attempted expense that needs a human repair, a receipt,
// or conversation.
//
// It knows nothing about where a Message came from. Source adapters (internal/telegram)
// produce Messages; this package consumes them. That split is the correctness property
// the T-75 plan rests on: the same text must mean the same thing whichever door it
// came through, so the parse lives here exactly once and never in an adapter.
//
// Parsing itself is NOT reimplemented here. A message is Converted only when the
// production boundary (internal/parse) accepts its fields, so a line this package emits
// is a line batch-auto will take, by construction rather than by agreement.
package capture

import (
	"fmt"
	"strings"
	"time"
)

// Attachment names the non-text payload that rode along with a message.
type Attachment int

const (
	NoAttachment Attachment = iota
	// FileAttachment is a document — in practice a PDF boleto.
	FileAttachment
	PhotoAttachment
)

// Message is one unit of the capture stream, whatever source produced it.
type Message struct {
	ID int
	// SentAt is when the human sent the message. Its calendar day is the year
	// evidence for a bare DD/MM date (plan D1); nothing downstream has it.
	SentAt time.Time
	// Text is the flattened plain text; "" when the message is only an attachment.
	Text       string
	Attachment Attachment
}

// Class is what a message turned out to be.
type Class int

const (
	// Converted: the text parsed as item;date;value and Outcome.Line is ready for batch-auto.
	Converted Class = iota
	// Rejected: an attempted expense the boundary would not take; Outcome.Err says why.
	Rejected
	// Receipt: an attachment with no text — a receipt for an expense that was typed
	// separately (user ruling, s75). Not missing spend, not an attempted expense.
	Receipt
	// Ignored: conversation. Counted so the export is fully accounted for, never repaired.
	Ignored
)

// Outcome is the verdict on one message.
type Outcome struct {
	Message Message
	Class   Class
	// Err is set for Rejected only: the boundary's own error, so a consumer can
	// errors.Is / errors.As it (parse.ErrInvalidDate, parse.ErrInvalidValue,
	// FieldCountError) instead of matching English text.
	Err error
	// Line is set for Converted only: item;DD/MM/YYYY;value, where value is the RAW
	// token the human typed — "900,00/3" and "405,25 x4" pass through unexpanded
	// (plan D2), because the installment count must survive into the review queue.
	Line string
	// Date is set for Converted only: the resolved expense date, which decides the
	// month file the line belongs to (plan D7) — not the day the message was sent.
	Date time.Time
	// Repair is set when the line came from a single-edit repair (plan D4): the
	// repaired text that parsed. Empty for a line that parsed exactly as typed, which
	// is what lets the report show repaired lines on their own — they deserve a glance.
	Repair string
}

// AmbiguousRepairError reports a text that MORE THAN ONE single edit would make
// parse (plan D4). Two readings means the tool would be guessing, and a wrong guess
// writes a budget row indistinguishable from a right one; a rejection costs one edit
// because the row comes back with this reason inline. Same asymmetry as T-64.
type AmbiguousRepairError struct {
	Readings []string
}

func (e AmbiguousRepairError) Error() string {
	return fmt.Sprintf("ambiguous: %d different single-edit repairs would parse: %s",
		len(e.Readings), strings.Join(e.Readings, " | "))
}

// FieldCountError reports a text that did not split into the three fields
// batch-auto reads. The message mirrors batch-auto's own, so the human learns
// one rule.
type FieldCountError struct {
	Got int
}

func (e FieldCountError) Error() string {
	return fmt.Sprintf("expected 3 fields (item;DD/MM;value), got %d", e.Got)
}

// Bucket is the report line an Outcome lands on. A rejection is refined by WHY it
// was rejected: "rejected" alone would hide exactly the drift the dry-run report
// exists to show — a message moving from "bad value" to "bad date" changes what
// repair it needs while leaving the rejected count untouched.
type Bucket string

const (
	BucketConverted Bucket = "converted"
	// BucketRepaired: Converted, but only after one edit (plan D4). Reported apart
	// from converted so a human can look at exactly the lines the tool touched.
	BucketRepaired  Bucket = "repaired"
	BucketReceipts  Bucket = "receipts"
	BucketIgnored   Bucket = "ignored"
	BucketAmbiguous Bucket = "rejected: ambiguous"
	BucketBadDate   Bucket = "rejected: bad date"
	BucketBadValue  Bucket = "rejected: bad value"
	// BucketRejectedOther is the fallback for a rejection that names neither field —
	// an empty item, for instance. It should stay rare; if it grows, split it.
	BucketRejectedOther Bucket = "rejected: other"
)

// Summary is the fold over a stream's outcomes that the dry-run report prints.
//
// MESSAGES AND LINES ARE DIFFERENT NUMBERS since plan D10, and conflating them is the
// easy mistake here: one message can be a list of ten expenses, so ten outcomes carry
// one id. Counts are about LINES (an outcome each), ids are about MESSAGES (a human act
// each), and the report says both rather than pretending they agree.
type Summary struct {
	// Total is how many MESSAGES were read — a split list counts once.
	Total int
	// Lines is how many expense lines were classified — a split list counts N.
	Lines int
	// Splits is how many messages turned out to be multi-line lists (D10).
	Splits int
	// Counts is LINES per bucket. It is not len(IDs[bucket]): a list whose lines
	// disagree puts one id in two buckets, and several lines in one.
	Counts map[Bucket]int
	// IDs lists the message ids that landed in each bucket, first-seen order, with no
	// repeat inside a bucket, so the report can be checked per id and not only per
	// count. An id in TWO buckets means a list whose lines disagreed.
	IDs map[Bucket][]int
}
