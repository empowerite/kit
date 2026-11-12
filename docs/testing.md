# Testing

This page says how to run the kit's own tests, how long they take, and why a Mac desk runs them slower than a
GitHub runner. Some of the numbers are still being measured; each one says where it came from.

## Running the tests

- `kit gate` runs [bin/ci](../bin/ci), as CI does. It runs `kit lint`, `kit syntax --check`, the schemas, the
  links, the bats suite, the Go tests in `projects/tui/`, and `kit check`.
- To run one file, use `kit tools run bats tests/<file>.bats`. It puts the pinned tools first on `PATH`.
- bats runs one job per CPU, on GNU parallel, which `tools:` in `kit.yml` pins.
- Each test runs apart from the desk. `helper:isolate` in [tests/helper.bash](../tests/helper.bash) moves `HOME`,
  `TMPDIR`, and the kit's cache under the test's own directory. It also clears what CI and a desk set, such as
  `GITHUB_ACTIONS`, `KIT_TOOLCHAIN`, and `MISE_ENV`, so a test reads the same in both places.

## Contracts

The kit's public surface has snapshots in the repository, and a test fails when the kit prints anything else. A
change to the surface is then a diff a review sees.

| What | Snapshot | Held by |
| --- | --- | --- |
| the verbs, each with its purpose and group: `kit syntax --list` | `tests/snapshots/verbs.tsv` | `tests/syntax.bats` |
| every verb's grammar: `kit syntax <verb>` | `projects/tui/internal/syntax/testdata/<verb>.json` | `tests/syntax.bats` |
| the form kittui draws from each grammar | `projects/tui/internal/form/testdata/<verb>.golden` | `go test` |
| the tools and the resources `kit serve` offers | `tests/snapshots/tools.json`, `tests/snapshots/resources.json` | `tests/serve.bats` |

`bin/contracts` writes them all afresh: it runs those tests with `KIT_SNAPSHOTS=write`, and the form's with
`-update`. Review `git diff`, and commit what changed with the change that changed it. `bin/ci` runs the suite with
`KIT_SNAPSHOTS` empty, so the gate never writes one.

`tests/claims.bats` holds what the pages say to the code: every tracked Markdown and MDX page but those under
`docs/superpowers/`, dated plans, specs, and reviews, and under `tests/` and a `testdata/`, which are fixtures.

- Every `kit <verb> …` in a code span, and on a line of a shell's fenced block, names a verb, and flags it takes, and
  is a command line its grammar takes, as the loader's guard judges one; a form such as `<path>` or `[--all]` is not
  judged past its verb.
- Every `kit.yml` key a page names is one the schema describes and `settings:validate` knows, and the table of the
  keys in [packages.md](packages.md#settings) names all of them.
- Every path a code span names from the root, or from `kit/`, is there.
- A count of what the code holds is the count: the stages `kit flow` draws, the checks `kit ci` runs, the seams, the
  tools of `kit serve` that read only and that run what the files name, the steps of `kit welcome`, and the tracker
  providers. A list of the stages is the stages, in their order, and the tables of the seams, of the tools, and of the
  jq modules name every one.

A page that shows a command meant to fail puts `<!-- kit:claims skip -->` on the line before it, and an MDX page
`{/* kit:claims skip */}`: nothing is judged up to the next blank line, or to the end of the fenced block that opens
next.

## Time limits

A test that waits forever must fail fast, not run until GitHub stops it. One did: a test started the real `kittui`
on a terminal, and run 41556526134 waited two hours before it was cancelled.

- Each test fails at 300 seconds. `bin/ci` sets `BATS_TEST_TIMEOUT`. The check of every governed file against its
  template took over two minutes at a desk, so it runs as eight shards of about 44 seconds each.
- bats counts down that limit in a background job of the test's own shell, so a test waits for the jobs it started
  by their process IDs, `wait "${pids[@]}"`. A bare `wait` waits for the countdown too, and the test fails at the
  limit however soon its own jobs end.
- The CI job fails at 35 minutes. `answers.ci_timeout_minutes` in `kit.yml` sets it; see
  [packages/github/ci.md](packages/github/ci.md). It is about twice the longest run.

## How long a run takes

| Where | Jobs | Tests | Job | Source |
| --- | --- | --- | --- | --- |
| GitHub runner, `ubuntu-latest`, 4 vCPUs | 1 | 19 to 27 minutes | 25 to 31 minutes | runs 41578395796, 41586980312 |
| GitHub runner, `ubuntu-latest`, 4 vCPUs | 4 | 15 minutes | 12 to 18 minutes | runs 41596868298 to 41599080269 |
| Mac desk, M2 Ultra, 24 cores | 1 | 46 to 58 minutes | | commit 40e3f42 |
| Mac desk, M2 Ultra, 24 cores | 24 | 7.5 minutes | | commit 40e3f42 |

The desk's serial times were taken while other test runs loaded the machine. At a desk, 12 jobs took 564 seconds and
48 took 630, so more jobs than CPUs only queue. On the runner, four jobs make each test about 2.6 times slower, so
running them side by side gains only about 1.3 times: the runner has no spare CPU.

The next saving, at a desk and in CI alike, is fewer process starts. Loading the yaml module runs `yq` three times,
and the template check loads it about 2,000 times.

## Why a Mac desk is slower

The Mac is the faster machine; macOS is the slower system. The same 17 tests of `tests/commit.bats`, run on one M2
Ultra, then on a GitHub runner:

| Where | Time | Against Linux on the Mac |
| --- | --- | --- |
| Linux, on the Mac, in Docker | 43 s | 1.0 |
| GitHub runner, `ubuntu-latest` | 86 s | 2.0 times |
| macOS, on the Mac | 126 s | 2.9 times |

Each of the 17 took 2.4 to 3.4 times as long on macOS as on Linux on the same hardware. Linux there beats the
runner by 2 times, so the hardware is faster; macOS spends that and more.

How far a desk falls behind a runner depends on what else runs. On a quiet Mac with its malware scanner off, the
desk took 1.5 times a runner's time. Earlier, with a backup and the scanner running, it took 2.4 times, over 130
tests; the median test took 2.5 times as long, and none was faster.

Nearly all of a test's time goes to starting short processes: bash, git, jq, and yq. A typical test, `kit commit
prints a warning and commits`, took 10.7 seconds at the desk against 4.3 on a runner:

- 3.5 seconds ran the test's own code
- 6.6 seconds ran in the kernel, starting processes
- 0.6 seconds waited on something else

Starting a process costs far more on macOS. On the same Mac, with little else running, each start took:

| Start | macOS | Linux, in Docker | macOS against Linux |
| --- | --- | --- | --- |
| a bash subshell | 0.80 ms | 0.21 ms | 3.8 times |
| `true` | 1.79 ms | 0.26 ms | 7 times |
| `git` | 5.56 ms | 0.45 ms | 12 times |

A start that runs a program costs the most, and `git`, which loads libraries, costs the most of those. On macOS,
about half of a `git` start is the new process's own work before `git` runs, and the rest is the kernel's.

### mise shims

A tool reached through a mise shim costs about 23 ms a call, against 3.5 ms for the tool itself. `kit gate` and
`kit tools run` put the tools' own directories first on `PATH`, as CI does. Running `bats` straight from a shell
that has only the shims on `PATH` makes the suite several times slower. With `--jobs`, it breaks: a shim rebuilds
`PATH` and drops bats' own `libexec`.

### Developer Tools

macOS checks each new executable the first time it runs: Gatekeeper and XProtect read it before it may start. The
tests write many small scripts, such as fakes and shims, so they pay that check often. Each check took about
22 ms. Measured with 200 scripts:

| Run from | The same script 200 times | 200 new scripts |
| --- | --- | --- |
| an app listed as a developer tool | 0.95 s | 1.00 s |
| an app not listed | 0.96 s | 5.40 s |

An app listed as a developer tool skips the check, and so does every process it starts. To list one:

1. Run `spctl developer-mode enable-terminal`. It adds a Developer Tools list to System Settings.
2. Add each app you run the tests from, in System Settings › Privacy & Security › Developer Tools. A terminal
   such as iTerm, and an editor such as VS Code, are the usual ones.
3. Quit and reopen each app. A process started before the change still pays the check.

The check is real but small. Waiting of every kind was 4 to 6 percent of the tests timed, so skipping the check
saves at most that much. It does not speed up starting a program that has run before: from iTerm, a developer tool,
a start took 1.0 ms for a subshell, 2.1 ms for `true`, and 6.0 ms for `git`, the same as from an app not listed.
Most of the gap is the cost of starting processes, above.

### Still to measure

- How much a malware scanner and a backup each add. They were turned off together, so the 2.4 and 1.5 above do
  not separate them.
