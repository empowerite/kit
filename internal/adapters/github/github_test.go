package github

import (
	"context"
	"strings"
	"testing"

	"github.com/empowerite/kit/internal/ports"
)

var (
	ctx                 = context.Background()
	_   ports.Releases  = Releases{}
	_   ports.Downloads = Downloads{}
)

const aRelease = `{"tag_name": "v0.11.0", "assets": [
  {"name": "a.tar.gz", "browser_download_url": "https://github.com/o/r/releases/download/v0.11.0/a.tar.gz", "digest": "sha256:abc"},
  {"name": "b.tar.gz", "browser_download_url": "https://github.com/o/r/releases/download/v0.11.0/b.tar.gz", "digest": null}]}`

func TestAssetGivesTheFilesAddressAndGitHubsDigest(t *testing.T) {
	m := memory(map[string]Answer{"https://api.github.com/repos/o/r/releases/tags/v0.11.0": {200, aRelease}})
	r := NewReleases("a-token", m)
	got, err := r.Asset(ctx, "o/r", "v0.11.0", "a.tar.gz")
	if err != nil || got != (ports.Asset{URL: "https://github.com/o/r/releases/download/v0.11.0/a.tar.gz", Digest: "sha256:abc"}) {
		t.Errorf("Asset = %+v, %v", got, err)
	}
	if got, _ := r.Asset(ctx, "o/r", "v0.11.0", "b.tar.gz"); got.Digest != "" {
		t.Errorf("Asset of a file GitHub keeps no digest of = %+v; want no digest", got)
	}
	if _, err := r.Asset(ctx, "o/r", "v0.11.0", "c.tar.gz"); err == nil || err.Error() != "github: o/r v0.11.0 has no file c.tar.gz" {
		t.Errorf("Asset of a file the aRelease does not hold gave %v", err)
	}
	if h := m.Headers(); h["Authorization"] != "Bearer a-token" || h["Accept"] != "application/vnd.github+json" {
		t.Errorf("Asset sent the headers %v", h)
	}
}

func TestReleasesSendNoTokenWhereThereIsNone(t *testing.T) {
	m := memory(map[string]Answer{"https://api.github.com/repos/o/r/releases/latest": {200, aRelease}})
	if tag, err := NewReleases("", m).Newest(ctx, "o/r"); tag != "v0.11.0" || err != nil {
		t.Errorf("Newest = %q, %v; want v0.11.0", tag, err)
	}
	if _, sent := m.Headers()["Authorization"]; sent {
		t.Error("Newest sent an Authorization header with no token")
	}
}

func TestReleasesSayWhatGitHubSays(t *testing.T) {
	m := memory(map[string]Answer{
		"https://api.github.com/repos/o/r/releases/tags/v9": {404, `{"message": "Not Found"}`},
		"https://api.github.com/repos/o/r/releases/latest":  {403, `{"message": "API rate limit exceeded"}`},
		"https://api.github.com/repos/o/r/releases/tags/v1": {200, `not json`},
	})
	r := NewReleases("", m)
	if _, err := r.Asset(ctx, "o/r", "v9", "a"); err == nil || err.Error() != "github: o/r has no release v9" {
		t.Errorf("Asset of a release that is not there gave %v", err)
	}
	if _, err := r.Newest(ctx, "o/r"); err == nil || !strings.Contains(err.Error(), "403: API rate limit exceeded") {
		t.Errorf("Newest gave %v; want what GitHub said", err)
	}
	if _, err := r.Asset(ctx, "o/r", "v1", "a"); err == nil || !strings.HasPrefix(err.Error(), "github: ") {
		t.Errorf("Asset of what is not JSON gave %v", err)
	}
	if _, err := r.Newest(ctx, "o/none"); err == nil || !strings.Contains(err.Error(), "nothing at") {
		t.Errorf("Newest of what memory does not hold gave %v", err)
	}
	if _, err := r.Asset(ctx, "o/none", "v1", "a"); err == nil {
		t.Error("Asset of what memory does not hold gave no error")
	}
}

func TestDownloadsGetWhatAnAddressHolds(t *testing.T) {
	m := memory(map[string]Answer{"https://x/a": {200, "bytes"}, "https://x/gone": {404, "Not Found"}})
	d := NewDownloads(m)
	if got, err := d.Get(ctx, "https://x/a"); string(got) != "bytes" || err != nil {
		t.Errorf("Get = %q, %v; want bytes", got, err)
	}
	if _, err := d.Get(ctx, "https://x/gone"); err == nil || err.Error() != "download: https://x/gone: 404" {
		t.Errorf("Get of what is gone gave %v", err)
	}
	if _, err := d.Get(ctx, "https://x/none"); err == nil {
		t.Error("Get of what memory does not hold gave no error")
	}
}

func TestTheRealClientOpensNothingForAnAddressNoMachineServes(t *testing.T) {
	// The address's scheme is one no client speaks, so the request ends before any connection opens. The nightly tier
	// reaches GitHub for real.
	for _, err := range []error{
		func() error { _, err := NewDownloads(nil).Get(ctx, "kit-none://x/a"); return err }(),
		func() error { _, err := NewDownloads(nil).Get(ctx, "::"); return err }(),
	} {
		if err == nil || !strings.HasPrefix(err.Error(), "download: ") {
			t.Errorf("Get gave %v; want a download error", err)
		}
	}
}
