package binary

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestRunWithTimeoutKillsSlowCommand(t *testing.T) {
	start := time.Now()
	err := runWithTimeout(exec.Command("sleep", "2"), 1)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("runWithTimeout error = %v, want timeout", err)
	}
	if elapsed := time.Since(start); elapsed > 1500*time.Millisecond {
		t.Fatalf("runWithTimeout elapsed = %s, want timeout near 1s", elapsed)
	}
}

func TestRunCaptureWithTimeoutReturnsOutput(t *testing.T) {
	out, err := runCaptureWithTimeout(exec.Command("printf", "ok"), 1)
	if err != nil {
		t.Fatalf("runCaptureWithTimeout error = %v", err)
	}
	if out != "ok" {
		t.Fatalf("output = %q, want ok", out)
	}
}
