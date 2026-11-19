# kit: keep it trustworthy

kit keeps the work of AI coding agents trustworthy. What it is for, and the principles it holds, are in the
[charter](docs/charter.md). How work is done here is in [CONTRIBUTING.md](CONTRIBUTING.md).

## Status

Pre-alpha. This repository holds those two pages, the [foundations design](docs/design/foundations.md), and one
command, `kit`, which says which commit it was built from: `go build ./cmd/kit`, then `./kit version`. `bin/gate`
command, `kit`, which says which commit it was built from: `go build ./cmd/kit`, then `./kit version`. `bin/gate`
runs the gate, `bin/forge` checks that GitHub holds what the gate leans on, and `bin/build` builds kit for macOS and
Linux. The design's packages are in place, and empty.

## License

MIT; see [LICENSE](LICENSE).
