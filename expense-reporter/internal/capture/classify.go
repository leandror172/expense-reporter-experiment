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
// The DATE itself is validated by the boundary (parse.Fields / parse.Date) on the
// returned string, so there is one set of year rules. What is validated HERE is the
// DISTANCE from the message, on both branches (D1 amendment, s75): the boundary's
// past side is unguarded, so an explicit `21/08/1200` would otherwise convert and
// open its own month file. Every error wraps parse.ErrInvalidDate.
func ResolveDateField(field string, sentAt time.Time) (string, error) {
	parts := strings.Split(strings.TrimSpace(field), "/")
	switch len(parts) {
	case 2:
		return resolveBareDate(parts[0], parts[1], sentAt)
	case 3:
		return resolveExplicitYear(parts[0], parts[1], parts[2], sentAt)
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
// (each trimmed, NOT zero-padded — that differs from the bare branch on purpose),
// with a 2-digit year expanded to 20YY and a 4-digit year kept. Any other year
// length is an error wrapping parse.ErrInvalidDate.
//
// The explicit year still WINS THE SELECTION — nothing here re-resolves which year
// the human meant. What was added by the s75 D1 amendment is the same window the
// bare branch already applies, used here to REJECT rather than to choose: the
// boundary allows years 1..9999 and only looks forward, so without this an absurd
// `21/08/1200` converts and the expense silently leaves its month. D5 means a line
// that parses as typed is never repaired, so it never meets D4's window check —
// this is the only place that gap can be closed.
//
// When day, month and year are not all integers forming a real calendar date, the
// window is SKIPPED: there is no date to measure, and the boundary owns that
// rejection. The distance is only meaningful once the date exists.
func resolveExplicitYear(dayStr, monthStr, yearStr string, sentAt time.Time) (string, error) {
	dayStr = strings.TrimSpace(dayStr)
	monthStr = strings.TrimSpace(monthStr)
	yearStr = strings.TrimSpace(yearStr)

	if len(yearStr) == 2 {
		yearStr = "20" + yearStr
	} else if len(yearStr) != 4 {
		return "", fmt.Errorf("%w: year %q must have 2 or 4 digits", parse.ErrInvalidDate, yearStr)
	}
	resolved := fmt.Sprintf("%s/%s/%s", dayStr, monthStr, yearStr)

	day, month, err := parseDayMonth(dayStr, monthStr)
	if err != nil {
		return "", err
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return "", fmt.Errorf("%w: year %q is not an integer", parse.ErrInvalidDate, yearStr)
	}

	date, exists := calendarDate(year, month, day)
	if !exists {
		return resolved, nil
	}
	if days := daysFrom(sentAt, date); days < -windowPast || days > windowFuture {
		midnight := time.Date(sentAt.Year(), sentAt.Month(), sentAt.Day(), 0, 0, 0, 0, time.UTC)
		return "", fmt.Errorf("%w: %s is not within %d days before or %d days after %s", parse.ErrInvalidDate, resolved, windowPast, windowFuture, midnight.Format("2006-01-02"))
	}
	return resolved, nil
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

// Classify decides what one message is (plan D3). Empty text is a Receipt when an
// attachment rode along, otherwise Ignored. Otherwise the text is tried AS TYPED
// first, and a line that parses is Converted and NEVER repaired — plan D5, the hard
// precondition that keeps a well-formed "900,00/3" from ever being "repaired" into
// something else. Only a text that fails as typed is a candidate for repair, and only
// if it is an attempted expense at all; conversation is Ignored before any repair is
// tried, so chatter never reaches the repair queue.
func Classify(m Message, now time.Time) Outcome {
	text := strings.TrimSpace(m.Text)
	if text == "" {
		return textlessOutcome(m)
	}
	parsed, err := parseLine(text, m.SentAt, now)
	if err == nil {
		return convertedOutcome(m, parsed)
	}
	if !isAttemptedExpense(text) {
		return Outcome{Message: m, Class: Ignored}
	}
	return repairedOrRejected(m, text, err, now)
}

// parseLine is THE predicate for a line: exactly three fields, a date the message
// can resolve (D1), a value the boundary accepts. The line as typed and every repair
// candidate go through this one function, so a repair can never be accepted on
// looser terms than the original, and the boundary's error comes back untouched so
// the reason survives into the rejects file.
func parseLine(text string, sentAt, now time.Time) (parse.ParsedExpense, error) {
	fields := strings.Split(text, ";")
	if len(fields) != 3 {
		return parse.ParsedExpense{}, FieldCountError{Got: len(fields)}
	}
	resolvedDate, err := ResolveDateField(fields[1], sentAt)
	if err != nil {
		return parse.ParsedExpense{}, err
	}
	return parse.Fields(fields[0], resolvedDate, fields[2], parse.Options{Now: now})
}

// textlessOutcome: an attachment with nothing typed is a receipt for an expense
// typed separately (user ruling, s75); nothing at all is conversation.
func textlessOutcome(m Message) Outcome {
	if m.Attachment != NoAttachment {
		return Outcome{Message: m, Class: Receipt}
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

// ClassifyAll expands each message into its expense lines (D10) and runs Classify on
// each, in stream order. A message that is not a multi-line list expands to itself, so
// this stays 1:1 for everything the 2025 corpus ever contained.
func ClassifyAll(msgs []Message, now time.Time) []Outcome {
	var outcomes []Outcome
	for _, msg := range msgs {
		for _, line := range expandLists(msg) {
			outcomes = append(outcomes, Classify(line, now))
		}
	}
	return outcomes
}

// expandLists is plan D10: a message with more than one non-empty line, EVERY one of
// them 3-field-shaped (exactly two ';'), is N messages — one per line, each keeping the
// parent's id, timestamp and attachment. Anything else is returned untouched.
//
// The test is the SEMICOLON COUNT and deliberately NOT whether the line parses. The
// message that forced this rule — a ten-line backlog list in the 2026-02 export — has two
// lines carrying a `29/12/26` year typo, so "every line parses" would have refused to
// split it and lost all ten expenses. Shape is the evidence that a human typed a LIST;
// each line's date and value are then judged by the same predicate as any other message.
//
// All-or-nothing is what stops a one-expense message with a chatty second line from being
// torn in half: that second line has no semicolons, so nothing splits.
func expandLists(m Message) []Message {
	var lines []string
	for _, line := range strings.Split(m.Text, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	if len(lines) < 2 {
		return []Message{m}
	}
	for _, line := range lines {
		if strings.Count(line, ";") != 2 {
			return []Message{m}
		}
	}

	expanded := make([]Message, len(lines))
	for i, line := range lines {
		expanded[i] = Message{ID: m.ID, SentAt: m.SentAt, Text: line, Attachment: m.Attachment}
	}
	return expanded
}

// Bucket is the report line the outcome lands on. A rejection is refined by which
// field the boundary named, or by the field count, so the report shows WHY.
func (o Outcome) Bucket() Bucket {
	switch o.Class {
	case Converted:
		if o.Repair != "" {
			return BucketRepaired
		}
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
	var ambiguous AmbiguousRepairError
	switch {
	case errors.As(err, &ambiguous):
		return BucketAmbiguous
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

// Summarize folds outcomes into per-bucket LINE counts and per-bucket MESSAGE ids, both
// in first-seen order. Buckets nobody landed in are absent, so the report prints only
// what happened.
//
// Since D10 an outcome is a LINE, not a message, and the two counts diverge: a list whose
// lines disagree puts its one id in two buckets, and several of its lines in one bucket.
// Total counts distinct ids, Splits counts the ids that produced more than one line.
func Summarize(outcomes []Outcome) Summary {
	sum := Summary{Lines: len(outcomes), Counts: make(map[Bucket]int), IDs: make(map[Bucket][]int)}

	linesPerID := make(map[int]int)
	inBucket := make(map[Bucket]map[int]bool)
	for _, outcome := range outcomes {
		bucket := outcome.Bucket()
		id := outcome.Message.ID
		sum.Counts[bucket]++
		linesPerID[id]++

		if inBucket[bucket] == nil {
			inBucket[bucket] = make(map[int]bool)
		}
		if !inBucket[bucket][id] {
			inBucket[bucket][id] = true
			sum.IDs[bucket] = append(sum.IDs[bucket], id)
		}
	}

	sum.Total = len(linesPerID)
	for _, lines := range linesPerID {
		if lines > 1 {
			sum.Splits++
		}
	}
	return sum
}
