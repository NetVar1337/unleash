package patches

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/VoidChecksum/void-patcher-cc/embed"
	"github.com/VoidChecksum/void-patcher-cc/internal/console"
)

// LoadPatches loads all *.json patch files from a directory, skipping retired
// and disabled patches. Returns patches sorted by filename.
func LoadPatches(patchDir string) ([]Patch, error) {
	entries, err := os.ReadDir(patchDir)
	if err != nil {
		return nil, fmt.Errorf("reading patch dir %s: %w", patchDir, err)
	}

	// Sort entries by name for deterministic ordering.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var patches []Patch
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		fpath := filepath.Join(patchDir, e.Name())
		data, err := os.ReadFile(fpath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s%s %s: read error — %v%s\n",
				console.R, console.CROSS, e.Name(), err, console.X)
			continue
		}

		var p Patch
		if err := json.Unmarshal(data, &p); err != nil {
			fmt.Fprintf(os.Stderr, "%s%s %s: invalid JSON — %v%s\n",
				console.R, console.CROSS, e.Name(), err, console.X)
			continue
		}

		if p.Disabled || p.Retired {
			continue
		}

		p.File = fpath
		patches = append(patches, p)
	}
	return patches, nil
}

// LoadPatchesForScan loads js_replace patches suitable for signature scanning.
// Patches with scan_signatures explicitly set to false are excluded.
// The __file annotation is set for autoheal support.
func LoadPatchesForScan(patchDir string, respectScanFlag bool) ([]Patch, error) {
	entries, err := os.ReadDir(patchDir)
	if err != nil {
		return nil, fmt.Errorf("reading patch dir %s: %w", patchDir, err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var out []Patch
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		fpath := filepath.Join(patchDir, e.Name())
		data, err := os.ReadFile(fpath)
		if err != nil {
			continue
		}

		var p Patch
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}

		if p.Type != "js_replace" {
			continue
		}
		if p.Retired {
			continue
		}
		if respectScanFlag && !p.ShouldScan() {
			continue
		}

		p.File = fpath
		out = append(out, p)
	}
	return out, nil
}

// LoadPatchesFromEmbed loads all patches from the embedded filesystem.
// Falls back gracefully if no embedded patches are available.
func LoadPatchesFromEmbed() ([]Patch, error) {
	return loadPatchesFromFS(embed.EmbeddedPatches, "patches")
}

// LoadPatchesForScanFromEmbed loads scan-suitable patches from the embedded filesystem.
func LoadPatchesForScanFromEmbed(respectScanFlag bool) ([]Patch, error) {
	return loadPatchesForScanFromFS(embed.EmbeddedPatches, "patches", respectScanFlag)
}

// IsRetired quickly checks whether a patch JSON file is marked retired
// without fully parsing the entire structure.
func IsRetired(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var stub struct {
		Retired bool `json:"retired"`
	}
	if err := json.Unmarshal(data, &stub); err != nil {
		return false
	}
	return stub.Retired
}

// loadPatchesFromFS loads patches from an fs.FS (embed or disk override).
func loadPatchesFromFS(fsys fs.FS, root string) ([]Patch, error) {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, fmt.Errorf("reading embedded patch dir: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var patches []Patch
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		fpath := root + "/" + e.Name()
		data, err := fs.ReadFile(fsys, fpath)
		if err != nil {
			continue
		}

		var p Patch
		if err := json.Unmarshal(data, &p); err != nil {
			fmt.Fprintf(os.Stderr, "%s%s %s: invalid JSON — %v%s\n",
				console.R, console.CROSS, e.Name(), err, console.X)
			continue
		}

		if p.Disabled || p.Retired {
			continue
		}

		p.File = "(embedded) " + e.Name()
		patches = append(patches, p)
	}
	return patches, nil
}

// loadPatchesForScanFromFS loads scan-suitable patches from an fs.FS.
func loadPatchesForScanFromFS(fsys fs.FS, root string, respectScanFlag bool) ([]Patch, error) {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, fmt.Errorf("reading embedded patch dir: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var out []Patch
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		fpath := root + "/" + e.Name()
		data, err := fs.ReadFile(fsys, fpath)
		if err != nil {
			continue
		}

		var p Patch
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}

		if p.Type != "js_replace" {
			continue
		}
		if p.Retired {
			continue
		}
		if respectScanFlag && !p.ShouldScan() {
			continue
		}

		p.File = "(embedded) " + e.Name()
		out = append(out, p)
	}
	return out, nil
}
