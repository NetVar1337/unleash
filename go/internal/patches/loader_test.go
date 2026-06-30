package patches

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPatchesForScanSkipsDisabledRetiredAndNonScannable(t *testing.T) {
	dir := t.TempDir()
	writePatchJSON(t, dir, "01-enabled.json", `{"id":"enabled","type":"js_replace","patches":[{"search":"a","replace":"b"}]}`)
	writePatchJSON(t, dir, "02-disabled.json", `{"id":"disabled","type":"js_replace","disabled":true,"patches":[{"search":"a","replace":"b"}]}`)
	writePatchJSON(t, dir, "03-retired.json", `{"id":"retired","type":"js_replace","retired":true,"patches":[{"search":"a","replace":"b"}]}`)
	writePatchJSON(t, dir, "04-no-scan.json", `{"id":"no-scan","type":"js_replace","scan_signatures":false,"patches":[{"search":"a","replace":"b"}]}`)
	writePatchJSON(t, dir, "05-settings.json", `{"id":"settings","type":"settings","settings":{"x":"y"}}`)

	got, err := LoadPatchesForScan(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "enabled" {
		t.Fatalf("LoadPatchesForScan IDs = %#v, want only enabled", got)
	}
	if got[0].File != filepath.Join(dir, "01-enabled.json") {
		t.Fatalf("File = %q, want source path", got[0].File)
	}
}

func writePatchJSON(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
