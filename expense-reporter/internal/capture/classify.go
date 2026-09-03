package capture

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"expense-reporter/internal/parse"
)

// The year window for a bare DD/MM, in days relative to the message's calendar
// day. Measured on the 2025 export: over 354 dated messages the distance ran from
// −131 to +3 days, 279 of them on the message's own day. The past side is generous
// because a purchase is often reported weeks later; the future side has no support
// beyond +3 (+7 leaves room for a scheduled bill). The load-bearing invariant is
// that windowPast+windowFuture is under 365, so at most ONE candidate year can fit.
const (
	windowPast   = 180
	windowFuture = 7
)

// moneyToken is what makes a non-conforming text an ATTEMPTED expense rather than
// conversation: digits, a comma or dot, exactly two digits not followed by another
// digit — or the literal R$. Plan D3: this keeps chatter out of the repair queue
// while a message that talks about money still reaches a human.
var moneyToken = regexp.MustCompile(`(\d+[.,]\d{2}(\D|$))|R\$`)

// ResolveDateField turns a message's date field into "DD/MM/YYYY" (plan D1). The
// converter OWNS the year: an explicit year in the text is the human's and wins
// outright; a bare DD/MM is resolved by proximity to the message's own send time.
// Validation is NOT done here — the boundary (parse.Fields / parse.Date) does that
// on the returned string, so there is one set of year rules. Every error wraps
// parse.ErrInvalidDate.
func ResolveDateField(field string, sentAt time.Time) (string, error) {
	parts := strings.Split(strings.TrimSpace(field), "/")
	switch len(parts) {
	case 2:
		return resolveBareDate(parts[0], parts[1], sentAt)
	case 3:
		return resolveExplicitYear(parts[0], parts[1], parts[2])
	default:
		return "", fmt.Errorf("%w: expected DD/MM or DD/MM/YYYY, got %q", parse.ErrInvalidDate, field)
	}
}

// resolveBareDate picks the ONE year in {sent−1, sent, sent+1} for which day/month
// is a real calendar date landing within [−windowPast, +windowFuture] days of the
// message's calendar day, and formats it "DD/MM/YYYY" with zero padding.
// Errors (all wrapping parse.ErrInvalidDate): day or month not an integer; no
// candidate year makes it a calendar date (31/02); or candidates exist but none is
// inside the window — that error names the window and the message date.
func resolveBareDate(dayStr, monthStr string, sentAt time.Time) (string, error) {
	day, month, err := parseDayMonth(dayStr, monthStr)
	if err != nil {
		return "", err
	}

	var anyCalendarDate bool
	for _, year := range []int{sentAt.Year() - 1, sentAt.Year(), sentAt.Year() + 1} {
		date, exists := calendarDate(year, month, day)
		if !exists {
			continue
		}
		anyCalendarDate = true

		days := daysFrom(sentAt, date)
		if days >= -windowPast && days <= windowFuture {
			return fmt.Sprintf("%02d/%02d/%04d", day, month, year), nil
		}
	}

	if !anyCalendarDate {
		return "", fmt.Errorf("%w: %02d/%02d is not a calendar date", parse.ErrInvalidDate, day, month)
	}

	midnight := time.Date(sentAt.Year(), sentAt.Month(), sentAt.Day(), 0, 0, 0, 0, time.UTC)
	return "", fmt.Errorf("%w: no year puts %02d/%02d within %d days before or %d days after %s", parse.ErrInvalidDate, day, month, windowPast, windowFuture, midnight.Format("2006-01-02"))
}

// resolveExplicitYear returns "day/month/year" as the human wrote day and month
// (each trimmed), with a 2-digit year expanded to 20YY and a 4-digit year kept.
// Any other year length is an error wrapping parse.ErrInvalidDate. No calendar
// check here — the boundary validates the returned string.
func resolveExplicitYear(dayStr, monthStr, yearStr string) (string, error) {
	dayStr = strings.TrimSpace(dayStr)
	monthStr = strings.TrimSpace(monthStr)
	yearStr = strings.TrimSpace(yearStr)

	if len(yearStr) == 2 {
		yearStr = "20" + yearStr
	} else if len(yearStr) != 4 {
		return "", fmt.Errorf("%w: year %q must have 2 or 4 digits", parse.ErrInvalidDate, yearStr)
	}

	return fmt.Sprintf("%s/%s/%s", dayStr, monthStr, yearStr), nil
}

// parseDayMonth reads the two integers of a bare date; a non-integer wraps
// parse.ErrInvalidDate and names which of the two it was.
func parseDayMonth(dayStr, monthStr string) (int, int, error) {
	day, err := strconv.Atoi(strings.TrimSpace(dayStr))
	if err != nil {
		return 0, 0, fmt.Errorf("%w: day %q is not an integer", parse.ErrInvalidDate, dayStr)
	}
	month, err := strconv.Atoi(strings.TrimSpace(monthStr))
	if err != nil {
		return 0, 0, fmt.Errorf("%w: month %q is not an integer", parse.ErrInvalidDate, monthStr)
	}
	return day, month, nil
}

// calendarDate returns midnight UTC of y/m/d and whether that day exists in that
// month of that year (time.Date normalizes 31/02 into March, which is how the
// check works).
func calendarDate(year int, month, day int) (time.Time, bool) {
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return date, date.Day() == day && date.Month() == time.Month(month) && month >= 1 && month <= 12
}

// daysFrom is the signed distance in whole days from the message's calendar day
// (midnight UTC of sentAt's date) to the candidate: negative when the candidate is
// earlier than the message, positive when later.
func daysFrom(sentAt, candidate time.Time) int {
	midnight := time.Date(sentAt.Year(), sentAt.Month(), sentAt.Day(), 0, 0, 0, 0, time.UTC)
	return int(candidate.Sub(midnight).Hours() / 24)
}

// Classify decides what one message is (plan D3). Empty text is a Receipt when
// an attachment rode along, otherwise Ignored. Text that does not split into
// exactly three fields is Rejected only if it is an attempted expense; otherwise
// it is conversation and Ignored. Three fields go through ResolveDateField (D1)
// and then the production boundary; whatever the boundary says stands, and its
// error is kept untouched so the reason survives.
func Classify(m Message, now time.Time) Outcome {
	text := strings.TrimSpace(m.Text)
	if text == "" {
		return textlessOutcome(m)
	}
	fields := strings.Split(text, ";")
	if len(fields) != 3 {
		return wrongFieldCountOutcome(m, text, len(fields))
	}
	resolvedDate, err := ResolveDateField(fields[1], m.SentAt)
	if err != nil {
		return rejectedOutcome(m, err)
	}
	parsed, err := parse.Fields(fields[0], resolvedDate, fields[2], parse.Options{Now: now})
	if err != nil {
		return rejectedOutcome(m, err)
	}
	return convertedOutcome(m, parsed)
}

// textlessOutcome: an attachment with nothing typed is a receipt for an expense
// typed separately (user ruling, s75); nothing at all is conversation.
func textlessOutcome(m Message) Outcome {
	if m.Attachment != NoAttachment {
		return Outcome{Message: m, Class: Receipt}
	}
	return Outcome{Message: m, Class: Ignored}
}

// wrongFieldCountOutcome rejects an attempted expense with the field count as the
// reason, and ignores anything else — the count, not the content, is the diagnosis.
func wrongFieldCountOutcome(m Message, text string, got int) Outcome {
	if isAttemptedExpense(text) {
		return rejectedOutcome(m, FieldCountError{Got: got})
	}
	return Outcome{Message: m, Class: Ignored}
}

// isAttemptedExpense is plan D3's rule: a semicolon or a money-shaped token.
func isAttemptedExpense(text string) bool {
	return strings.Contains(text, ";") || moneyToken.MatchString(text)
}

func rejectedOutcome(m Message, err error) Outcome {
	return Outcome{Message: m, Class: Rejected, Err: err}
}

// convertedOutcome carries the RAW value token into the line (plan D2): the
// installment count must reach the review queue, so nothing is expanded here.
func convertedOutcome(m Message, parsed parse.ParsedExpense) Outcome {
	return Outcome{
		Message: m,
		Class:   Converted,
		Line:    fmt.Sprintf("%s;%s;%s", parsed.Item, parsed.DateString(), parsed.RawValue),
		Date:    parsed.Date,
	}
}

// ClassifyAll runs Classify on each message, in stream order.
func ClassifyAll(msgs []Message, now time.Time) []Outcome {
	outcomes := make([]Outcome, len(msgs))
	for i, msg := range msgs {
		outcomes[i] = Classify(msg, now)
	}
	return outcomes
}

// Bucket is the report line the outcome lands on. A rejection is refined by which
// field the boundary named, or by the field count, so the report shows WHY.
func (o Outcome) Bucket() Bucket {
	switch o.Class {
	case Converted:
		return BucketConverted
	case Receipt:
		return BucketReceipts
	case Ignored:
		return BucketIgnored
	default:
		return rejectionBucket(o.Err)
	}
}

func rejectionBucket(err error) Bucket {
	var fce FieldCountError
	switch {
	case errors.As(err, &fce):
		return fieldCountBucket(fce.Got)
	case errors.Is(err, parse.ErrInvalidDate):
		return BucketBadDate
	case errors.Is(err, parse.ErrInvalidValue):
		return BucketBadValue
	default:
		return BucketRejectedOther
	}
}

func fieldCountBucket(got int) Bucket {
	if got == 1 {
		return Bucket("rejected: 1 field")
	}
	return Bucket(fmt.Sprintf("rejected: %d fields", got))
}

// Summarize folds outcomes into per-bucket id lists in stream order. Buckets
// nobody landed in are absent, so the report prints only what happened.
func Summarize(outcomes []Outcome) Summary {
	sum := Summary{Total: len(outcomes), IDs: make(map[Bucket][]int)}
	for _, outcome := range outcomes {
		bucket := outcome.Bucket()
		sum.IDs[bucket] = append(sum.IDs[bucket], outcome.Message.ID)
	}
	return sum
}
