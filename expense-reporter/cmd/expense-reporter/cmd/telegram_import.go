package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"expense-reporter/internal/batch"
	"expense-reporter/internal/capture"
	"expense-reporter/internal/telegram"
)

var (
	telegramImportDryRun bool
	telegramImportOutDir string
	telegramImportForce  bool
)

// rejectsFileName is deliberately NOT expenses-*.csv: a glob over the month files must
// never pick up rows that still need a human. It re-runs through batch-auto only after
// each line has been repaired in place (T-63 shape).
const rejectsFileName = "telegram-rejects.csv"

var telegramImportCmd = &cobra.Command{
	Use:   "telegram-import <result.json>",
	Short: "Convert a Telegram chat export into batch-auto month CSVs",
	Long: `Reads a Telegram Desktop "Export chat history" result.json and turns the
expenses typed into the group into item;DD/MM/YYYY;value lines for batch-auto.

No model, no network, no workbook. Every message lands in exactly one bucket:
converted, rejected (an attempted expense that needs a human repair), receipts
(an attachment beside an expense that was typed separately), or ignored
(conversation). --dry-run prints that bucket report and writes nothing.`,
	Args: cobra.ExactArgs(1),
	RunE: runTelegramImport,
}

func init() {
	rootCmd.AddCommand(telegramImportCmd)
	telegramImportCmd.Flags().BoolVar(&telegramImportDryRun, "dry-run", false,
		"Classify every message and print the bucket report; write nothing")
	telegramImportCmd.Flags().StringVar(&telegramImportOutDir, "out-dir", "",
		"Directory for the month files (expenses-YYYY-MM.csv) and "+rejectsFileName+"; required unless --dry-run")
	telegramImportCmd.Flags().BoolVar(&telegramImportForce, "force", false,
		"Overwrite month/rejects files that already exist in --out-dir (refused otherwise)")
}

func runTelegramImport(cmd *cobra.Command, args []string) error {
	msgs, err := telegram.ReadExport(args[0])
	if err != nil {
		return err
	}
	outcomes := capture.ClassifyAll(msgs, time.Now())
	summary := capture.Summarize(outcomes)
	if telegramImportDryRun {
		writeBucketReport(cmd.OutOrStdout(), "telegram-import: dry run, nothing written", summary)
		return nil
	}
	if telegramImportOutDir == "" {
		return fmt.Errorf("telegram-import writes files: pass --out-dir DIR (the close's files live outside the repo), or --dry-run to only report")
	}
	written, err := writeOutputs(telegramImportOutDir, telegramImportForce, capture.MonthFiles(outcomes), capture.Rejects(outcomes))
	if err != nil {
		return err
	}
	writeBucketReport(cmd.OutOrStdout(), fmt.Sprintf("telegram-import: wrote %d file(s) to %s", len(written), telegramImportOutDir), summary)
	writeFilesReport(cmd.OutOrStdout(), written)
	return nil
}

// writtenFile is one output file and its body line count, for the report.
type writtenFile struct {
	Name  string
	Lines int
}

// writeOutputs writes every month file and, when there are rejects, the rejects file
// into dir — ALL of them or NONE (plan D8). Before the first write it checks that dir
// exists and, unless force is set, that no target file already exists; any blocker
// aborts the whole run with every blocking name in the error and "--force" named as
// the way out. The default output directory holds the hand-repaired files of the first
// real close, which a silent overwrite would destroy. Month files get a one-line '#'
// provenance header (batch-auto's reader skips '#' lines and blank lines); the rejects
// file goes through batch.WriteFailedRows so the T-63 shape has exactly one producer.
// Returns what was written, in the order written.
func writeOutputs(dir string, force bool, months []capture.MonthFile, rejects []capture.Reject) ([]writtenFile, error) {
	if err := refuseUnlessWritable(dir, force, targetNames(months, rejects)); err != nil {
		return nil, err
	}

	written, err := writeMonthFiles(dir, months)
	if err != nil {
		return nil, err
	}

	if len(rejects) > 0 {
		rejectedRows := asFailedRows(rejects)
		path := filepath.Join(dir, rejectsFileName)
		if err = batch.WriteFailedRows(path, rejectedRows); err != nil {
			return nil, err
		}
		written = append(written, writtenFile{Name: rejectsFileName, Lines: len(rejectedRows)})
	}

	return written, nil
}

// targetNames lists every file this run would create: the month files, plus the
// rejects file when there is at least one reject (none otherwise — its existence is
// the signal, T-63).
func targetNames(months []capture.MonthFile, rejects []capture.Reject) []string {
	names := make([]string, len(months))
	for i, m := range months {
		names[i] = m.Name
	}
	if len(rejects) > 0 {
		names = append(names, rejectsFileName)
	}
	return names
}

// existingFiles returns the names in names that already exist in dir, in the given order.
func existingFiles(dir string, names []string) []string {
	var blockers []string
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			blockers = append(blockers, name)
		}
	}
	return blockers
}

// monthFileBody renders one month file: the header line, then every line, each ending
// in "\n". Header, exactly:
//
//	# <Name>: expenses dated that month, converted by telegram-import. Text after '   # ' on a line is a comment batch-auto ignores.
func monthFileBody(f capture.MonthFile) string {
	var b strings.Builder
	b.Grow(1024)
	b.WriteString("# ")
	b.WriteString(f.Name)
	b.WriteString(": expenses dated that month, converted by telegram-import. Text after '   # ' on a line is a comment batch-auto ignores.\n")
	for _, line := range f.Lines {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// asFailedRows adapts the converter's rejects to batch's row type, Text → OriginalLine
// and Reason → Reason.
func asFailedRows(rejects []capture.Reject) []batch.FailedRow {
	if len(rejects) == 0 {
		return nil
	}
	rows := make([]batch.FailedRow, len(rejects))
	for i, r := range rejects {
		rows[i] = batch.FailedRow{OriginalLine: r.Text, Reason: r.Reason}
	}
	return rows
}

// writeFilesReport prints the files block after the bucket report. Exact shape
// ("%d line" singular for 1, "%d lines" otherwise; the rejects file gets the trailing
// instruction):
//
//	files:
//	  expenses-2025-01.csv   1 line
//	  expenses-2025-05.csv   2 lines
//	  telegram-rejects.csv   4 lines — repair each in place, then run the file through batch-auto
func writeFilesReport(w io.Writer, files []writtenFile) {
	if len(files) == 0 {
		return
	}
	fmt.Fprintln(w, "  files:")
	for _, f := range files {
		unit := "line"
		if f.Lines != 1 {
			unit = "lines"
		}
		suffix := ""
		if f.Name == rejectsFileName {
			suffix = " — repair each in place, then run the file through batch-auto"
		}
		fmt.Fprintf(w, "    %-22s %d %s%s\n", f.Name, f.Lines, unit, suffix)
	}
}

// refuseUnlessWritable checks that dir is an existing directory and, unless force
// is set, that no target files already exist in it.
func refuseUnlessWritable(dir string, force bool, names []string) error {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("--out-dir %s is not an existing directory", dir)
	}
	if !force {
		blockers := existingFiles(dir, names)
		if len(blockers) > 0 {
			return fmt.Errorf("refusing to overwrite %s in %s — pass --force to replace", strings.Join(blockers, ", "), dir)
		}
	}
	return nil
}

// writeMonthFiles writes all month files into dir.
func writeMonthFiles(dir string, months []capture.MonthFile) ([]writtenFile, error) {
	var written []writtenFile
	for _, m := range months {
		path := filepath.Join(dir, m.Name)
		body := monthFileBody(m)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", m.Name, err)
		}
		written = append(written, writtenFile{Name: m.Name, Lines: len(m.Lines)})
	}
	return written, nil
}

// writeBucketReport prints the headline and then the bucket report: one line per
// bucket with its count, and — for every bucket but converted — the message ids that
// landed in it, so the report can be checked per id and not only per count (a message
// moving between two buckets leaves every count unchanged). The headline is the
// caller's because it says what the run DID ("dry run, nothing written" vs "wrote N
// file(s) to …"); the buckets are the same either way. Exact shape, which the
// acceptance scenario pins:
//
//	telegram-import: dry run, nothing written
//	  12 messages
//	   4 converted
//	   2 receipts (attachments beside typed expenses, skipped): 3 4
//	   1 rejected: 2 fields: 5
//	   1 rejected: bad date: 8
//	   1 ignored (conversation): 6
//
// Order: converted, receipts, then every "rejected: …" bucket sorted by name, then
// ignored. Counts are right-aligned in a 4-wide column. Converted, receipts and
// ignored print even when zero; a rejected bucket exists only if something landed
// in it.
func writeBucketReport(w io.Writer, headline string, s capture.Summary) {
	fmt.Fprintln(w, headline)
	fmt.Fprintf(w, "%4d messages\n", s.Total)
	fmt.Fprintf(w, "%4d converted\n", len(s.IDs[capture.BucketConverted]))
	writeBucketLine(w, "repaired (one edit made the line parse)", s.IDs[capture.BucketRepaired])
	writeBucketLine(w, "receipts (attachments beside typed expenses, skipped)", s.IDs[capture.BucketReceipts])
	for _, bucket := range sortedRejectedBuckets(s.IDs) {
		writeBucketLine(w, string(bucket), s.IDs[bucket])
	}
	writeBucketLine(w, "ignored (conversation)", s.IDs[capture.BucketIgnored])
}

// writeBucketLine writes "%4d label" followed by ": id id …" when there are ids.
func writeBucketLine(w io.Writer, label string, ids []int) {
	if len(ids) == 0 {
		fmt.Fprintf(w, "%4d %s\n", 0, label)
		return
	}
	fmt.Fprintf(w, "%4d %s: %s\n", len(ids), label, joinIDs(ids))
}

// sortedRejectedBuckets lists the "rejected: …" buckets present in a summary, by
// name, so the report is stable run to run whatever order the map iterates in.
func sortedRejectedBuckets(idMap map[capture.Bucket][]int) []capture.Bucket {
	var rejected []capture.Bucket
	for bucket := range idMap {
		if strings.HasPrefix(string(bucket), "rejected: ") {
			rejected = append(rejected, bucket)
		}
	}
	sort.Slice(rejected, func(i, j int) bool { return rejected[i] < rejected[j] })
	return rejected
}

func joinIDs(ids []int) string {
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = strconv.Itoa(id)
	}
	return strings.Join(strs, " ")
}
