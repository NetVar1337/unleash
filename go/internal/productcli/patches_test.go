package productcli

import (
	"testing"
	"testing/fstest"

	"github.com/VoidChecksum/unleash/internal/bytepatch"
)

func TestLoadPatchesFromFSAndMergeOverride(t *testing.T) {
	fsys := fstest.MapFS{
		"patches/01-base.json":     {Data: []byte(`{"id":"base","patches":[{"search":"a","replace":"b"}]}`)},
		"patches/02-disabled.json": {Data: []byte(`{"id":"disabled","disabled":true,"patches":[{"search":"a","replace":"b"}]}`)},
	}
	loaded, err := LoadPatchesFromFS(fsys, "patches")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].ID != "base" {
		t.Fatalf("loaded = %#v, want only enabled base patch", loaded)
	}
	extra := []bytepatch.Patch{{ID: "base", Patches: []bytepatch.SubPatch{{Search: "new", Replace: "old"}}}, {ID: "extra"}}
	merged := MergePatches(loaded, extra)
	if len(merged) != 2 {
		t.Fatalf("merged length = %d, want 2", len(merged))
	}
	if merged[0].Patches[0].Search != "new" {
		t.Fatalf("override not applied: %#v", merged[0])
	}
	if merged[1].ID != "extra" {
		t.Fatalf("extra patch not appended: %#v", merged)
	}
}
