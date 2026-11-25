package control

import "testing"

func TestAppNotRunningErrorUsesRootyardName(t *testing.T) {
	if got, want := (AppNotRunningError{}).Error(), "rootyard desktop app is not running"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}
