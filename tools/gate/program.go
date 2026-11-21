package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
)

// program judges a program kit fetched: its sha256 is the one kit.lock names for the tool on this machine's
// platform. It gives 0 when it is, 1 when it is not, and 2 when it cannot judge.
func program(options map[string]string, stdout io.Writer) int {
	path, tool, lockPath := options["--program"], options["--tool"], options["--lock"]
	if len(options) != 3 || path == "" || tool == "" || lockPath == "" {
		fmt.Fprintln(stdout, "usage: gate --program=<path> --tool=<name> --lock=<kit.lock>")
		return 2
	}
	cannot := func(err error) int {
		fmt.Fprintf(stdout, "gate: %s: %v\n", tool, err)
		return 2
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return cannot(err)
	}
	text, err := os.ReadFile(lockPath)
	if err != nil {
		return cannot(err)
	}
	var lock struct {
		Tools map[string]struct {
			Files map[string]struct {
				ProgramDigest string `json:"program_digest"`
			} `json:"files"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(text, &lock); err != nil {
		return cannot(fmt.Errorf("kit.lock: %w", err))
	}
	platform := runtime.GOOS + "/" + runtime.GOARCH
	want := lock.Tools[tool].Files[platform].ProgramDigest
	if want == "" {
		return cannot(fmt.Errorf("kit.lock holds no file of %s for %s", tool, platform))
	}
	sum := sha256.Sum256(body)
	if got := "sha256:" + hex.EncodeToString(sum[:]); got != want {
		fmt.Fprintf(stdout, "gate: %s: refused: %s has the digest %s, and kit.lock names %s\n", tool, path, got, want)
		return 1
	}
	fmt.Fprintf(stdout, "gate: %s: ok: %s is the program kit.lock names\n", tool, path)
	return 0
}
