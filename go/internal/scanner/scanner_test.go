package scanner

import (
	"testing"

	"github.com/VoidChecksum/unleash/internal/patches"
)

func TestEmptyAnchorsDoNotPanic(t *testing.T) {
	s := NewSigScanner("alpha beta gamma")
	if off := s.FindAnchor([]string{""}, 400); off != nil {
		t.Fatalf("FindAnchor empty = %v, want nil", *off)
	}
	if offs := s.AllOccurrences(""); len(offs) != 0 {
		t.Fatalf("AllOccurrences empty = %v, want empty", offs)
	}
	if off, method, confidence := s.FindAnchorFuzzy([]string{"", "   "}, 400); off != nil || method != "none" || confidence != 0 {
		t.Fatalf("FindAnchorFuzzy blank = %v %q %f, want nil none 0", off, method, confidence)
	}
	if off, confidence := s.FindAnchorNgram([]string{""}, 400); off != nil || confidence != 0 {
		t.Fatalf("FindAnchorNgram blank = %v %f, want nil 0", off, confidence)
	}
	if off, confidence := s.FindAnchorLevenshtein([]string{""}, 400); off != nil || confidence != 0 {
		t.Fatalf("FindAnchorLevenshtein blank = %v %f, want nil 0", off, confidence)
	}
	if off, confidence := s.FindAnchorStructural([]string{""}, 400); off != nil || confidence != 0 {
		t.Fatalf("FindAnchorStructural blank = %v %f, want nil 0", off, confidence)
	}
}

func TestScanPatchesIgnoresBlankAnchors(t *testing.T) {
	s := NewSigScanner("prefix live-anchor suffix")
	rows := s.ScanPatches([]patches.Patch{
		{
			ID:            "blank-anchor",
			Type:          "js_replace",
			AnchorStrings: []string{"", "live-anchor"},
			Patches:       []patches.SubPatch{{Search: "missing", Replace: "present"}},
		},
	})
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0].Status != "ok" || rows[0].Method != "anchor" {
		t.Fatalf("row = %#v, want ok via anchor", rows[0])
	}
}
