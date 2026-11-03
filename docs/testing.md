# Testing

This page says how to run the kit's own tests, how long they take, and why a Mac desk runs them slower than a
GitHub runner. Some of the numbers are still being measured; each one says where it came from.

## Running the tests

- `kit gate` runs [bin/ci](../bin/ci), as CI does. It runs `kit lint`, `kit syntax --check`, the schemas, the
  links, the bats suite, the Go tests in `projects/tui/`, and `kit check`.
- To run one file, use `kit tools run bats tests/<file>.bats`. It puts the pinned tools first on `PATH`.
- bats runs one job per CPU, on GNU parallel, which `mise.toml` pins.
- Each test runs apart from the desk. `helper:isolate` in [tests/helper.bash](../tests/helper.bash) moves `HOME`,
  `TMPDIR`, and the kit's cache under the test's own directory. It also clears what CI and a desk set, such as
  `GITHUB_ACTIONS`, `KIT_TOOLCHAIN`, and `MISE_ENV`, so a test reads the same in both places.

## Time limits

A test that waits forever must fail fast, not run until GitHub stops it. One did: a test started the real `kittui`
on a terminal, and run 41556526134 waited two hours before it was cancelled.

- Each test fails at 300 seconds. `bin/ci` sets `BATS_TEST_TIMEOUT`. The check of every governed file against its
  template took over two minutes at a desk, so it runs as eight shards of about 44 seconds each.
- The CI job fails at 35 minutes. `ci_timeout_minutes` in `kit.yml` sets it; see
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

Test for test, a Mac desk takes about 2.4 times as long as a runner. Over 130 tests timed in both places, the
median test took 2.5 times as long, and none was faster. A slowdown that even points at the machine, not at any
one test.

Nearly all of a test's time goes to starting short processes: bash, git, jq, and yq. A typical test, `kit commit
prints a warning and commits`, took 10.7 seconds at the desk against 4.3 on a runner:

- 3.5 seconds ran the test's own code
- 6.6 seconds ran in the kernel, starting processes
- 0.6 seconds waited on something else

Starting a process costs more on macOS. Measured on the desk, each start took:

| Start | macOS, desk | Linux, same Mac, in Docker |
| --- | --- | --- |
| a bash subshell | 1.0 ms | 1.2 ms |
| `true` | 2.1 ms | 1.3 ms |
| `git` | 6.4 ms | 1.7 ms |

The Linux numbers are preliminary: they were taken while the Mac was busy. A clean comparison is still to come.

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

- A clean comparison of macOS and Linux on the same Mac, with nothing else running.
- How much a malware scanner or a backup running at the same time adds.
