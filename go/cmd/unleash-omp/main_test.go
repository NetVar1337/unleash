package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VoidChecksum/unleash/internal/omp"
)

func TestHelpListsUnleashOMPCommands(t *testing.T) {
	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"Unleash-OMP", "setup", "patch", "status", "verify", "install-rules", "uninstall-rules", "rollback"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help missing %q in:\n%s", want, out)
		}
	}
}

func TestVerifyReadsPIConfigDir(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("PI_CONFIG_DIR", configRoot)
	agentDir := filepath.Join(configRoot, "agent")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "config.yml"), []byte("tools:\n  approvalMode: yolo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := runVerify(buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "config: ok") {
		t.Fatalf("verify did not read PI_CONFIG_DIR config:\n%s", buf.String())
	}
}

func TestBundledOMPPatchesCoverPolicyAndTelemetry(t *testing.T) {
	patches, err := loadOMPPatches()
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	total := 0
	for _, p := range patches {
		ids[p.ID] = true
		total += len(p.Patches)
	}
	for _, want := range []string{
		"omp-telemetry-endpoints-localhost",
		"omp-approval-policy-allowall",
		"omp-acp-permission-gate-disable",
	} {
		if !ids[want] {
			t.Fatalf("missing bundled patch %q in %#v", want, ids)
		}
	}
	if total < 6 {
		t.Fatalf("bundled OMP subpatches = %d, want at least 6", total)
	}
}

func TestOMPPatchDirMergesAndOverridesBundledPatches(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "01-extra.json"), []byte(`{"id":"extra-omp","patches":[{"search":"a","replace":"b"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "02-override.json"), []byte(`{"id":"bundled","patches":[{"search":"new","replace":"old"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	base := []omp.Patch{{ID: "bundled", Patches: []omp.SubPatch{{Search: "old", Replace: "new"}}}}
	extra, err := loadOMPPatchesFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	merged := mergeOMPPatches(base, extra)
	ids := map[string]omp.Patch{}
	for _, p := range merged {
		ids[p.ID] = p
	}
	if _, ok := ids["extra-omp"]; !ok {
		t.Fatalf("extra patch missing from merged set: %#v", ids)
	}
	if ids["bundled"].Patches[0].Search != "new" {
		t.Fatalf("override did not replace bundled patch: %#v", ids["bundled"])
	}
}
