package toolchain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/empowerite/kit/internal/ports"
)

// digestOf gives a sha256 as a lock writes it: "sha256:<hex>".
func digestOf(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Resolve locks a tool at a version: for each platform it downloads the file, takes its digest, holds it to the
// digest GitHub keeps and to the checksum the tool publishes, where there are any, and takes the program's own
// digest. A digest that nothing confirms is refused, unless unconfirmed.
func Resolve(ctx context.Context, releases ports.Releases, downloads ports.Downloads, name, version string, e Entry,
	unconfirmed bool) (Locked, error) {
	l := Locked{Version: version, Entry: e, Files: map[Platform]LockedFile{}}
	tag := expand(e.Tag, version, "")
	for _, p := range Platforms {
		file := expand(e.Files[p], version, "")
		refuse := func(format string, a ...any) (Locked, error) {
			return Locked{}, fmt.Errorf("%s %s for %s: %s", name, version, p, fmt.Sprintf(format, a...))
		}
		asset, err := releases.Asset(ctx, e.GitHub, tag, file)
		if err != nil {
			return refuse("%v", err)
		}
		body, err := downloads.Get(ctx, asset.URL)
		if err != nil {
			return refuse("%v", err)
		}
		got := digestOf(body)
		confirmed := []string{}
		if asset.Digest != "" {
			if asset.Digest != got {
				return refuse("%s was downloaded with the digest %s, and GitHub keeps %s", file, got, asset.Digest)
			}
			confirmed = append(confirmed, "github")
		}
		if e.Checksums != "" {
			sums := expand(e.Checksums, version, file)
			published, err := publishedSum(ctx, releases, downloads, e.GitHub, tag, sums, file)
			switch {
			case err != nil:
				return refuse("%v", err)
			case published != got:
				return refuse("%s was downloaded with the digest %s, and %s publishes %s", file, got, name, published)
			}
			confirmed = append(confirmed, "published")
		}
		if len(confirmed) == 0 && !unconfirmed {
			return refuse("nothing confirms the digest of %s: GitHub keeps none, and %s publishes none. "+
				"--unconfirmed pins it on kit's own download alone", file, name)
		}
		program := expand(e.Program, version, file)
		bytes, err := extract(file, body, program)
		if err != nil {
			return refuse("%v", err)
		}
		l.Files[p] = LockedFile{asset.URL, got, program, digestOf(bytes), confirmed}
	}
	return l, nil
}

// publishedSum gives the checksum a tool publishes of a file, from the release's file of checksums: a line that holds
// a sha256 alone, or a sha256 and the file's name.
func publishedSum(ctx context.Context, releases ports.Releases, downloads ports.Downloads, repo, tag, sums,
	file string) (string, error) {
	asset, err := releases.Asset(ctx, repo, tag, sums)
	var text []byte
	if err == nil {
		text, err = downloads.Get(ctx, asset.URL)
	}
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(text), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && len(fields[0]) == 64 &&
			(len(fields) == 1 || strings.TrimPrefix(fields[len(fields)-1], "*") == file) {
			if _, err := hex.DecodeString(fields[0]); err == nil {
				return "sha256:" + strings.ToLower(fields[0]), nil
			}
		}
	}
	return "", fmt.Errorf("%s holds no checksum of %s", sums, file)
}
