package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpListsUnleashGPTCommands(t *testing.T) {
	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"Unleash-GPT", "setup", "patch", "status", "verify", "install-rules", "uninstall-rules", "rollback"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help missing %q in:\n%s", want, out)
		}
	}
}
