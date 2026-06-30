package cmd

import "testing"

func TestLimitedCountHelpersHonorCount(t *testing.T) {
	if got := bytesCountN([]byte("aa aa aa"), []byte("aa"), 1); got != 1 {
		t.Fatalf("bytesCountN limited = %d, want 1", got)
	}
	if got := bytesCountN([]byte("aa aa aa"), []byte("aa"), 0); got != 3 {
		t.Fatalf("bytesCountN all = %d, want 3", got)
	}
	if got := regexCountMatchesN("aa", []byte("aa aa aa"), 1); got != 1 {
		t.Fatalf("regexCountMatchesN limited = %d, want 1", got)
	}
	if got := regexCountMatchesN("aa", []byte("aa aa aa"), 0); got != 3 {
		t.Fatalf("regexCountMatchesN all = %d, want 3", got)
	}
}
