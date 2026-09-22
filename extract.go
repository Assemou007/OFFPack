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

func safeArchivePath(destDir, archiveName string) (string, error) {
	name := filepath.ToSlash(archiveName)
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "\x00") {
		return "", fmt.Errorf("chemin d'archive invalide: %q", archiveName)
	}

	relative := stripTopDir(name)
	if relative == "" || relative == "." {
		return "", nil
	}
	if relative == ".." || strings.HasPrefix(relative, "../") || strings.Contains(relative, "/../") {
		return "", fmt.Errorf("path traversal détecté: %q", archiveName)
	}

	root, err := filepath.Abs(destDir)
	if err != nil {
		return "", err
	}
	target, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		return "", err
	}
	if target != root && !strings.HasPrefix(target, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("chemin hors destination: %q", archiveName)
	}
	return target, nil
}

func extractTarball(tarballPath, destDir string) error {
	archiveFile, err := os.Open(tarballPath)
	if err != nil {
		return fmt.Errorf("impossible d'ouvrir %s: %w", tarballPath, err)
	}
	defer archiveFile.Close()

	gz, err := gzip.NewReader(archiveFile)
	if err != nil {
		return fmt.Errorf("erreur de décompression: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
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
		if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeLink {
			return fmt.Errorf("liens tar non autorisés: %q", header.Name)
		}

		target, err := safeArchivePath(destDir, header.Name)
		if err != nil {
			return err
		}
		if target == "" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)&0755); err != nil {
				return err
			}
		case tar.TypeReg:
			output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode)&0755)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(output, tr)
			closeErr := output.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		}
	}

	return nil
}

func stripTopDir(path string) string {
	parts := strings.SplitN(filepath.ToSlash(path), "/", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}
