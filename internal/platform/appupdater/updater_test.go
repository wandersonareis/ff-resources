//go:build !server

package appupdater

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

func TestPrefixAssetMatcherPicksOwnApp(t *testing.T) {
	m := prefixAssetMatcher("pulse")
	assets := []github.ReleaseAsset{
		{Name: "spira-2.5.0-portable-windows-amd64.zip"},
		{Name: "pulse-2.5.0-portable-windows-amd64.zip"},
		{Name: "ff-resources-suite.zip"},
	}
	idx := m(updater.CheckRequest{Platform: "windows", Arch: "amd64"}, assets)
	if idx != 1 {
		t.Fatalf("índice = %d, quer 1 (pulse)", idx)
	}
}

func TestPrefixAssetMatcherIgnoresOtherAppAndPlatform(t *testing.T) {
	m := prefixAssetMatcher("pulse")
	assets := []github.ReleaseAsset{
		{Name: "spira-2.5.0-portable-windows-amd64.zip"},
		{Name: "pulse-2.5.0-portable-darwin-arm64.zip"},
		{Name: "ff-resources-suite.zip"},
	}
	if idx := m(updater.CheckRequest{Platform: "windows", Arch: "amd64"}, assets); idx != -1 {
		t.Fatalf("índice = %d, quer -1 (não há pulse windows/amd64)", idx)
	}
}

type fakeProvider struct {
	rel *updater.Release
	err error
}

func (f fakeProvider) Name() string { return "fake" }

func (f fakeProvider) Check(context.Context, updater.CheckRequest) (*updater.Release, error) {
	return f.rel, f.err
}

func (fakeProvider) Download(context.Context, *updater.Release, io.Writer, func(int64, int64)) error {
	return nil
}

func TestNoAssetIsUpToDate(t *testing.T) {
	p := noAssetIsUpToDate{inner: fakeProvider{
		err: errors.New("github: release v2.5.0 has no asset for windows/amd64"),
	}}
	rel, err := p.Check(context.Background(), updater.CheckRequest{})
	if err != nil {
		t.Fatalf("no asset deveria virar nil/nil: %v", err)
	}
	if rel != nil {
		t.Fatalf("release = %+v, quer nil", rel)
	}

	p = noAssetIsUpToDate{inner: fakeProvider{err: errors.New("network down")}}
	if _, err := p.Check(context.Background(), updater.CheckRequest{}); err == nil {
		t.Fatal("erro que não é de asset deveria propagar")
	}
}
