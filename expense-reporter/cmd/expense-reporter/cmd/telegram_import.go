package cmd

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"expense-reporter/internal/capture"
	"expense-reporter/internal/telegram"
)

var telegramImportDryRun bool

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
}

func runTelegramImport(cmd *cobra.Command, args []string) error {
	msgs, err := telegram.ReadExport(args[0])
	if err != nil {
		return err
	}
	outcomes := capture.ClassifyAll(msgs, time.Now())
	if !telegramImportDryRun {
		return fmt.Errorf("writing month CSVs is not implemented yet (T-75 step 3); run with --dry-run")
	}
	writeBucketReport(cmd.OutOrStdout(), capture.Summarize(outcomes))
	return nil
}

// writeBucketReport prints the dry-run report: one line per bucket with its count,
// and — for every bucket but converted — the message ids that landed in it, so the
// report can be checked per id and not only per count (a message moving between
// two buckets leaves every count unchanged). Exact shape, which the acceptance
// scenario pins:
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
func writeBucketReport(w io.Writer, s capture.Summary) {
	fmt.Fprintln(w, "telegram-import: dry run, nothing written")
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
