package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func extractTarball(tarballPath, destDir string) error {
	f, err := os.Open(tarballPath)
	if err != nil {
		return fmt.Errorf("impossible d'ouvrir %s: %w", tarballPath, err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("erreur de décompression: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("erreur de lecture du tar: %w", err)
		}

		relativePath := stripTopDir(header.Name)
		if relativePath == "" {
			continue
		}

		targetPath := filepath.Join(destDir, relativePath)

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			fw, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(fw, tr); err != nil {
				fw.Close()
				return err
			}
			fw.Close()
		}
	}

	return nil
}

func stripTopDir(path string) string {
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}
