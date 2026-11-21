// Command kit keeps the work of AI coding agents trustworthy. Its commands, their words, options, and help, are data:
// kit's commands.yml. This file gives each command its handler, and wires the real systems in.
package main

import (
	"context"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/empowerite/kit/internal/adapters/git"
	"github.com/empowerite/kit/internal/adapters/github"
	kitos "github.com/empowerite/kit/internal/adapters/os"
	"github.com/empowerite/kit/internal/cli"
	"github.com/empowerite/kit/internal/data"
	"github.com/empowerite/kit/internal/toolchain"
)

// build gives what Go recorded of this build. A test stands in for it.
var build = debug.ReadBuildInfo

// exit ends the program with a status. A test stands in for it.
var exit = os.Exit

// commands gives kit's commands, from its data. A test stands in for it.
var commands = cli.Commands

// handlers does each command, by its words joined by a space.
var handlers = both(map[string]cli.Handler{
	"version":   version,
	"data show": dataShow,
}, cli.Tools(desk))

// both gives the handlers of both maps.
func both(a, b map[string]cli.Handler) map[string]cli.Handler {
	maps.Copy(a, b)
	return a
}

// desk gives the machine kit runs on, with the real systems wired in.
func desk() (toolchain.Desk, error) {
	dir, err := os.Getwd()
	token := os.Getenv("GH_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	return toolchain.Desk{
		Dir:       dir,
		Files:     kitos.Files{},
		Git:       git.New(kitos.Processes{}),
		Releases:  github.NewReleases(token, nil),
		Downloads: github.NewDownloads(nil),
		Presence:  kitos.NewPresence(),
		Machine:   kitos.Machine{},
		Processes: kitos.Processes{},
	}, err
}

func main() {
	if named := filepath.Base(os.Args[0]); !strings.HasPrefix(named, "kit") {
		// kit started by a tool's name, through a link in .kit/bin.
		exit(tool(os.Args[0], os.Args[1:], os.Stderr))
		return
	}
	exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// tool runs the tool kit was started as, in kit's place, once it holds to the lock. Run so, kit ends and the tool
// goes on in its process, so tool returns only where the tool could not run, and says why.
func tool(argv0 string, args []string, stderr io.Writer) int {
	d, err := desk()
	if err == nil {
		err = toolchain.RunTool(context.Background(), d, argv0, args)
	}
	fmt.Fprintf(stderr, "kit: %s: %v\n", filepath.Base(argv0), err)
	return 127
}

// run does what the words after kit ask, and gives the exit status.
func run(args []string, stdout, stderr io.Writer) int {
	all, err := commands()
	if err != nil {
		fmt.Fprintf(stderr, "kit: its commands: %v\n", err)
		return 2
	}
	return cli.Run(context.Background(), all, handlers, args, stdout, stderr)
}

// version says which build this is.
func version(_ context.Context, _ cli.Call, out cli.Out) int {
	b := which(build())
	if out.JSON {
		return cli.Print(out, "version", b)
	}
	fmt.Fprintln(out.Stdout, b)
	return 0
}

// dataShow prints one of kit's own data files.
func dataShow(_ context.Context, call cli.Call, out cli.Out) int {
	name := call.Args[0]
	text, err := data.Show(name)
	if err != nil {
		fmt.Fprintf(out.Stderr, "kit: %v\n", err)
		return 1
	}
	if out.JSON {
		return cli.Print(out, "data", map[string]string{"name": name, "text": string(text)})
	}
	out.Stdout.Write(text)
	return 0
}

// Build is which build this is: the commit it was built from, when that commit was made, and whether the tree held
// changes the commit does not. Go records these in every build that go build makes, with no flag.
//
// A build does not say its release, since it does not know one: a release is a tag put on a commit whose build
// exists already, and two releases can name one build. Which releases name a commit is read from the tags.
type Build struct {
	Commit   string `json:"commit"`
	Time     string `json:"time"`
	Modified bool   `json:"modified"`
}

func (b Build) String() string {
	if b.Commit == "" {
		return "kit, a build that names no commit"
	}
	said := fmt.Sprintf("kit, built from commit %s of %s", b.Commit, b.Time)
	if b.Modified {
		said += ", and from changes it does not hold"
	}
	return said
}

// which reads which build this is from what Go recorded of it.
func which(info *debug.BuildInfo, recorded bool) Build {
	of := map[string]string{}
	if recorded {
		for _, setting := range info.Settings {
			of[setting.Key] = setting.Value
		}
	}
	return Build{of["vcs.revision"], of["vcs.time"], of["vcs.modified"] == "true"}
}
