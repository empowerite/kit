package main

import (
	"bytes"
	"os"
	"testing"
)

func TestRun(t *testing.T) {
	for _, c := range []struct {
		args     []string
		code     int
		out, err string
	}{
		{[]string{"version"}, 0, "unreleased\n", ""},
		{nil, 2, "", "usage: kit version\n"},
		{[]string{"version", "extra"}, 2, "", "usage: kit version\n"},
		{[]string{"flow"}, 2, "", "kit: no such command: flow\n"},
	} {
		var out, err bytes.Buffer
		if code := run(c.args, &out, &err); code != c.code || out.String() != c.out || err.String() != c.err {
			t.Errorf("run(%q) = %d, %q, %q; want %d, %q, %q", c.args, code, out.String(), err.String(), c.code, c.out, c.err)
		}
	}
}

func TestMainExitsWithWhatRunGives(t *testing.T) {
	args, was := os.Args, exit
	defer func() { os.Args, exit = args, was }()
	got := -1
	os.Args, exit = []string{"kit"}, func(code int) { got = code }
	main()
	if got != 2 {
		t.Errorf("main() exited %d with no command; want 2", got)
	}
}
