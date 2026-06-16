package main

import (
	"strconv"
	"strings"
)

// lower lowercases a sheet name for label use ("Fixas" -> "fixas").
func lower(s string) string { return strings.ToLower(s) }

// atoi parses a base-10 int, returning 0 on error (positions are always valid here).
func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// sumList joins cell refs into an Excel SUM of discrete terms: SUM(a,b,c).
func sumList(terms []string) string {
	return "SUM(" + strings.Join(terms, ",") + ")"
}

// sumRange returns SUM(first:last), collapsing to SUM(first) when first==last (golden style).
func sumRange(first, last string) string {
	if first == last {
		return "SUM(" + first + ")"
	}
	return "SUM(" + first + ":" + last + ")"
}
