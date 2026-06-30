package productcli

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/VoidChecksum/unleash/internal/bytepatch"
)

type patchEnvelope struct {
	bytepatch.Patch
	Disabled bool `json:"disabled,omitempty"`
	Retired  bool `json:"retired,omitempty"`
}

func LoadPatchesFromFS(fsys fs.FS, dir string) ([]bytepatch.Patch, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	out := make([]bytepatch.Patch, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		path := dir + "/" + entry.Name()
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		var patch patchEnvelope
		if err := json.Unmarshal(data, &patch); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		if patch.Disabled || patch.Retired {
			continue
		}
		out = append(out, patch.Patch)
	}
	return out, nil
}

func LoadPatchesFromDir(dir string) ([]bytepatch.Patch, error) {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return LoadPatchesFromFS(os.DirFS(filepath.Dir(dir)), filepath.Base(dir))
}

func MergePatches(base, extra []bytepatch.Patch) []bytepatch.Patch {
	merged := append([]bytepatch.Patch(nil), base...)
	index := make(map[string]int, len(merged))
	for i, p := range merged {
		index[p.ID] = i
	}
	for _, p := range extra {
		if i, ok := index[p.ID]; ok {
			merged[i] = p
			continue
		}
		index[p.ID] = len(merged)
		merged = append(merged, p)
	}
	return merged
}
