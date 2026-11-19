# kit: keep it trustworthy

kit keeps the work of AI coding agents trustworthy. What it is for, and the principles it holds, are in the
[charter](docs/charter.md). How work is done here is in [CONTRIBUTING.md](CONTRIBUTING.md). What the owner has
decided is in [docs/decisions.md](docs/decisions.md).

## Status

Pre-alpha. Beside those pages, this repository holds its first design and the retrospective it starts from, under
[docs/design/](docs/design/), and one command, `kit`, which says which commit it was built from:
`go build ./cmd/kit`, then `./kit version`. `bin/gate` runs the gate. The design's packages are in place, and empty.

## License

MIT; see [LICENSE](LICENSE).
