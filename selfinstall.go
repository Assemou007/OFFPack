package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func cmdSelfInstall() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: impossible de déterminer le chemin du binaire: %v\n", err)
		os.Exit(1)
	}

	destDir, err := installDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: impossible de créer %s: %v\n", destDir, err)
		os.Exit(1)
	}

	dest := filepath.Join(destDir, "offpack")
	if runtime.GOOS == "windows" {
		dest += ".exe"
	}

	data, err := os.ReadFile(exe)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur de lecture du binaire: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(dest, data, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur d'écriture vers %s: %v\n", dest, err)
		os.Exit(1)
	}

	fmt.Printf("✓ offpack installé dans %s\n", dest)
	fmt.Println("Assurez-vous que ce dossier est dans votre PATH.")
}

func installDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("impossible de trouver le répertoire home: %w", err)
		}
		return filepath.Join(home, ".local", "bin"), nil
	case "darwin":
		return "/usr/local/bin", nil
	case "linux":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("impossible de trouver le répertoire home: %w", err)
		}
		localBin := filepath.Join(home, ".local", "bin")
		if info, err := os.Stat(localBin); err == nil && info.IsDir() {
			return localBin, nil
		}
		if info, err := os.Stat("/usr/local/bin"); err == nil && info.IsDir() {
			return "/usr/local/bin", nil
		}
		return localBin, nil
	default:
		return "", fmt.Errorf("système d'exploitation non supporté: %s", runtime.GOOS)
	}
}
