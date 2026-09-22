package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBestVersionRanges(t *testing.T) {
	versions := []string{"0.1.0", "0.2.5", "1.2.0", "1.2.9", "1.9.0", "2.0.0"}
	tests := map[string]string{"^1.2.0": "1.9.0", "~1.2.0": "1.2.9", ">=1.2.0 <2.0.0": "1.9.0", "1.x": "1.9.0", "*": "2.0.0", "0.x": "0.2.5", "1.2.0 - 1.2.9": "1.2.9", "^0.2.0": "0.2.5"}
	for c, want := range tests {
		if got := resolveBestVersion(versions, c); got != want {
			t.Errorf("%s: got %s, want %s", c, got, want)
		}
	}
}
func TestExtractTarballRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	tarPath := filepath.Join(dir, "bad.tgz")
	f, _ := os.Create(tarPath)
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "package/../../outside", Mode: 0644, Size: 1})
	_, _ = tw.Write([]byte("x"))
	tw.Close()
	gz.Close()
	f.Close()
	if err := extractTarball(tarPath, filepath.Join(dir, "dest")); err == nil {
		t.Fatal("expected path traversal error")
	}
}
func TestExtractTarballExtractsPackageContent(t *testing.T) {
	dir := t.TempDir()
	tarPath := filepath.Join(dir, "ok.tgz")
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	data := []byte("ok")
	_ = tw.WriteHeader(&tar.Header{Name: "package/index.js", Mode: 0644, Size: int64(len(data))})
	_, _ = tw.Write(data)
	tw.Close()
	gz.Close()
	os.WriteFile(tarPath, b.Bytes(), 0644)
	dest := filepath.Join(dir, "dest")
	if err := extractTarball(tarPath, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "index.js")); err != nil {
		t.Fatal(err)
	}
}
