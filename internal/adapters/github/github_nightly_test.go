//go:build nightly

package github

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
)

func TestShellchecksReleaseForReal(t *testing.T) {
	r := NewReleases(os.Getenv("GH_TOKEN"), nil)
	a, err := r.Asset(ctx, "koalaman/shellcheck", "v0.11.0", "shellcheck-v0.11.0.linux.x86_64.tar.gz")
	if err != nil || a.Digest != "sha256:b7af85e41cc99489dcc21d66c6d5f3685138f06d34651e6d34b42ec6d54fe6f6" {
		t.Fatalf("Asset = %+v, %v; want GitHub's digest of the file", a, err)
	}
	body, err := NewDownloads(nil).Get(ctx, a.URL)
	sum := sha256.Sum256(body)
	if err != nil || "sha256:"+hex.EncodeToString(sum[:]) != a.Digest {
		t.Errorf("Get gave %d bytes, %v; want the file GitHub keeps the digest of", len(body), err)
	}
}
