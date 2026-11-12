package shell

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
)

// lookupEnv and goos are the environment and the system Browse reads; tests set them.
var (
	lookupEnv = os.LookupEnv
	goos      = runtime.GOOS
)

// ErrNoDisplay says Browse did not run an opener, as the person is not at the machine that runs kit: a screen gives
// the link to open by hand, and does not claim that a page opened.
var ErrNoDisplay = errors.New("no browser on this desk")

// away says whether the person is not at the machine that runs kit: over SSH, an opener would open the page on the
// other machine, and on Linux with neither an X nor a Wayland display, xdg-open has nowhere to open it. A variable that
// is empty is not set.
func away() bool {
	set := func(name string) bool { v, ok := lookupEnv(name); return ok && v != "" }
	if set("SSH_CONNECTION") || set("SSH_TTY") {
		return true
	}
	return goos == "linux" && !set("DISPLAY") && !set("WAYLAND_DISPLAY")
}

// Browse opens a link in the browser, as start does, without waiting for it; where the person is not at the machine, it
// runs no opener and returns ErrNoDisplay, so the screen gives the link.
func Browse(url string) error {
	if away() {
		return ErrNoDisplay
	}
	_, err := start(url)
	return err
}

// start starts the opener on a link: open on macOS, xdg-open elsewhere.
func start(url string) (*exec.Cmd, error) {
	opener := "xdg-open"
	if goos == "darwin" {
		opener = "open"
	}
	cmd := exec.Command(opener, url)
	return cmd, cmd.Start()
}
