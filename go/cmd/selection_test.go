package cmd

import (
	"testing"

	"github.com/VoidChecksum/unleash/internal/patches"
)

func TestFilterPatchesBySelectionAllMode(t *testing.T) {
	list := []patches.Patch{
		{ID: "a"}, {ID: "b"}, {ID: "c"},
	}
	s := selectionState{Mode: "all", Disabled: []string{"b"}}
	got := filterPatchesBySelection(list, s)
	if len(got) != 2 || got[0].ID != "a" || got[1].ID != "c" {
		t.Fatalf("got %#v", got)
	}
}

func TestFilterPatchesBySelectionOnlyMode(t *testing.T) {
	list := []patches.Patch{
		{ID: "a"}, {ID: "b"}, {ID: "c"},
	}
	s := selectionState{Mode: "only", Enabled: []string{"b"}}
	got := filterPatchesBySelection(list, s)
	if len(got) != 1 || got[0].ID != "b" {
		t.Fatalf("got %#v", got)
	}
}

func TestFilterPatchesByOnlyExcept(t *testing.T) {
	list := []patches.Patch{
		{ID: "a"}, {ID: "b"}, {ID: "c"},
	}
	got, err := filterPatchesByOnlyExcept(list, []string{"a", "c"}, nil)
	if err != nil || len(got) != 2 {
		t.Fatalf("only: %#v err=%v", got, err)
	}
	got, err = filterPatchesByOnlyExcept(list, nil, []string{"b"})
	if err != nil || len(got) != 2 || got[0].ID != "a" || got[1].ID != "c" {
		t.Fatalf("except: %#v err=%v", got, err)
	}
	_, err = filterPatchesByOnlyExcept(list, []string{"a"}, []string{"b"})
	if err == nil {
		t.Fatal("expected error for only+except")
	}
}

func TestResolvePatchIDsPrefix(t *testing.T) {
	list := []patches.Patch{
		{ID: "js-aup-refusal"},
		{ID: "js-aup-refusal-2"},
		{ID: "js-metrics-disable"},
	}
	ids, err := resolvePatchIDs(list, []string{"js-metrics"})
	if err != nil || len(ids) != 1 || ids[0] != "js-metrics-disable" {
		t.Fatalf("ids=%v err=%v", ids, err)
	}
	_, err = resolvePatchIDs(list, []string{"js-aup"})
	if err == nil {
		t.Fatal("expected ambiguous error")
	}
}

func TestParseIDList(t *testing.T) {
	got := parseIDList("a,b", "c d")
	if len(got) != 4 {
		t.Fatalf("got %v", got)
	}
}
