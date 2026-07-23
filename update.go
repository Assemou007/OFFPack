package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func cmdUpdate() {
	cd, err := cacheDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		os.Exit(1)
	}

	entries, err := os.ReadDir(cd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur de lecture du cache: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Println("Le cache est vide. Rien à mettre à jour.")
		return
	}

	var packages []string
	for _, e := range entries {
		if e.IsDir() {
			packages = append(packages, e.Name())
		}
	}

	fmt.Printf("Vérification des mises à jour pour %d packages...\n\n", len(packages))

	cacheSeen = make(map[string]bool)
	start := time.Now()
	updated := 0
	errors := 0

	for i, name := range packages {
		realName := unsanitizeName(name)
		fmt.Printf("[%d/%d] %s\n", i+1, len(packages), realName)

		currentVersion, err := latestCachedVersion(realName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ATTENTION: %v\n", err)
			errors++
			continue
		}

		meta, err := fetchPackageMeta(realName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ÉCHEC: %v\n", err)
			errors++
			continue
		}

		latestVersion := meta.DistTags.Latest
		if latestVersion == "" {
			continue
		}

		if latestVersion == currentVersion {
			continue
		}

		fmt.Printf("  Nouvelle version disponible: %s → %s\n", currentVersion, latestVersion)
		fmt.Printf("  Téléchargement de %s@%s...\n", realName, latestVersion)

		if err := cachePackage(realName, "latest"); err != nil {
			fmt.Fprintf(os.Stderr, "  ÉCHEC mise à jour: %v\n", err)
			errors++
		} else {
			updated++
		}
	}

	elapsed := time.Since(start)
	fmt.Println()
	fmt.Printf("Terminé en %v\n", elapsed.Round(time.Second))
	fmt.Printf("Packages mis à jour: %d\n", updated)

	if errors > 0 {
		fmt.Printf("Erreurs: %d\n", errors)
	}

	cdPath, _ := cacheDir()
	totalSize := dirSize(cdPath)
	fmt.Printf("Taille du cache: %s\n", formatSize(totalSize))
}

func latestCachedVersion(name string) (string, error) {
	sanitized := sanitizeName(name)
	cd, err := cacheDir()
	if err != nil {
		return "", err
	}

	pkgDir := filepath.Join(cd, sanitized)
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return "", err
	}

	var versions []string
	for _, e := range entries {
		if e.IsDir() {
			if sv, ok := parseSemver(e.Name()); ok {
				_ = sv
				versions = append(versions, e.Name())
			}
		}
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("aucune version trouvée pour %s", name)
	}

	best := findLatestVersion(versions)
	return best, nil
}
