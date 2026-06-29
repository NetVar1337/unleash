package updater

import "testing"

func TestUpdaterUsesUnleashRepository(t *testing.T) {
	if Repo != "VoidChecksum/unleash" {
		t.Fatalf("Repo = %q, want VoidChecksum/unleash", Repo)
	}
}
