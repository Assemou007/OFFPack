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
	for _, entry := range entries {
		if entry.IsDir() {
			packages = append(packages, entry.Name())
		}
	}
	fmt.Printf("Vérification des mises à jour pour %d packages...\n\n", len(packages))

	resetCacheSeen()
	start := time.Now()
	updated, errors := 0, 0
	for i, name := range packages {
		realName := unsanitizeName(name)
		fmt.Printf("[%d/%d] %s\n", i+1, len(packages), realName)
		currentVersion, err := latestCachedVersion(realName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ATTENTION: %v\n", err)
			errors++
			continue
		}
		meta, err := withMeta(realName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ÉCHEC: %v\n", err)
			errors++
			continue
		}
		latestVersion := meta.DistTags.Latest
		if latestVersion == "" || latestVersion == currentVersion {
			continue
		}
		fmt.Printf("  Nouvelle version disponible: %s → %s\n", currentVersion, latestVersion)
		if err := cachePackage(realName, "latest"); err != nil {
			fmt.Fprintf(os.Stderr, "  ÉCHEC mise à jour: %v\n", err)
			errors++
		} else {
			updated++
		}
	}

	fmt.Println()
	fmt.Printf("Terminé en %v\n", time.Since(start).Round(time.Second))
	fmt.Printf("Packages mis à jour: %d\n", updated)
	if errors > 0 {
		fmt.Printf("Erreurs: %d\n", errors)
	}
	cdPath, _ := cacheDir()
	fmt.Printf("Taille du cache: %s\n", formatSize(dirSize(cdPath)))
}

func latestCachedVersion(name string) (string, error) {
	cd, err := cacheDir()
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(filepath.Join(cd, sanitizeName(name)))
	if err != nil {
		return "", err
	}
	var versions []string
	for _, entry := range entries {
		if entry.IsDir() {
			if _, ok := parseSemver(entry.Name()); ok {
				versions = append(versions, entry.Name())
			}
		}
	}
	if len(versions) == 0 {
		return "", fmt.Errorf("aucune version trouvée pour %s", name)
	}
	return findLatestVersion(versions), nil
}
