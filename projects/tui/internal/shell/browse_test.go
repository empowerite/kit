package shell

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setEnv sets the environment and the system Browse reads for the test.
func setEnv(t *testing.T, env map[string]string, system string) {
	t.Helper()
	was, wasGoos := lookupEnv, goos
	lookupEnv = func(name string) (string, bool) { v, ok := env[name]; return v, ok }
	goos = system
	t.Cleanup(func() { lookupEnv, goos = was, wasGoos })
}

// openers puts an open and an xdg-open on PATH that record the link they are given in the file returned.
func openers(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	record := filepath.Join(dir, "record")
	for _, opener := range []string{"open", "xdg-open"} {
		script := "#!/bin/sh\nprintf '%s\\n' \"$1\" > " + record + "\n"
		if err := os.WriteFile(filepath.Join(dir, opener), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	return record
}

// Browse runs the opener with the link, and does not wait for it; with no opener, it fails.
func TestBrowseRunsTheOpener(t *testing.T) {
	record := openers(t)
	setEnv(t, map[string]string{}, "darwin")
	cmd, err := start("https://example.test/?kit=1")
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(record); string(got) != "https://example.test/?kit=1\n" {
		t.Errorf("the opener was given %q", got)
	}
	setEnv(t, map[string]string{}, "linux")
	if _, err := start("https://example.test/"); err != nil {
		t.Errorf("xdg-open: %v", err)
	}
	setEnv(t, map[string]string{}, "darwin")
	t.Setenv("PATH", t.TempDir())
	if err := Browse("https://example.test/"); err == nil || errors.Is(err, ErrNoDisplay) {
		t.Errorf("no opener on PATH: %v", err)
	}
}

// Over SSH, or on Linux with no display, Browse does not run the opener, and says there is no display here so that the
// screen gives the link; at a desk it runs the opener.
func TestBrowseDoesNotRunTheOpenerAwayFromTheDesk(t *testing.T) {
	record := openers(t)
	cases := []struct {
		name   string
		env    map[string]string
		system string
		away   bool
	}{
		{"ssh connection", map[string]string{"SSH_CONNECTION": "1.2.3.4 5 6.7.8.9 22"}, "darwin", true},
		{"ssh connection on linux with a display", map[string]string{"SSH_CONNECTION": "x", "DISPLAY": ":0"}, "linux", true},
		{"ssh tty", map[string]string{"SSH_TTY": "/dev/pts/0"}, "linux", true},
		{"linux without a display", map[string]string{}, "linux", true},
		{"linux with an empty display", map[string]string{"DISPLAY": ""}, "linux", true},
		{"linux with x", map[string]string{"DISPLAY": ":0"}, "linux", false},
		{"linux with wayland", map[string]string{"WAYLAND_DISPLAY": "wayland-0"}, "linux", false},
		{"macOS", map[string]string{}, "darwin", false},
		{"macOS with empty ssh values", map[string]string{"SSH_CONNECTION": "", "SSH_TTY": ""}, "darwin", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			os.Remove(record)
			setEnv(t, c.env, c.system)
			err := Browse("https://example.test/")
			for range 100 {
				if c.away {
					break
				}
				if _, statErr := os.Stat(record); statErr == nil {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			_, statErr := os.Stat(record)
			if c.away && (!errors.Is(err, ErrNoDisplay) || statErr == nil) {
				t.Errorf("err %v, opener ran %v", err, statErr == nil)
			}
			if !c.away && (err != nil || statErr != nil) {
				t.Errorf("err %v, opener ran %v", err, statErr == nil)
			}
		})
	}
}
