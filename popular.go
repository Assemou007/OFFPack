package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func cmdFetchPopular() {
	count := 500
	if len(os.Args) > 2 {
		parsed, err := strconv.Atoi(os.Args[2])
		if err == nil && parsed > 0 {
			count = parsed
		}
	}
	if count > len(topPackages) {
		count = len(topPackages)
	}

	packages := topPackages[:count]
	fmt.Printf("Téléchargement de %d packages populaires et leurs dépendances...\n", len(packages))
	fmt.Println("(Cela peut prendre du temps selon le nombre et la taille des packages)")
	fmt.Println()

	resetCacheSeen()
	start := time.Now()
	totalErrors := 0
	totalCached := 0
	for i, name := range packages {
		fmt.Printf("[%d/%d] %s\n", i+1, len(packages), name)
		if err := cachePackage(name, "latest"); err != nil {
			fmt.Fprintf(os.Stderr, "  ÉCHEC: %v\n", err)
			totalErrors++
		} else {
			totalCached++
		}
	}

	fmt.Println()
	fmt.Printf("Terminé en %v\n", time.Since(start).Round(time.Second))
	fmt.Printf("Packages populaires téléchargés: %d\n", totalCached)
	fmt.Printf("Package-versions dans le cache:   %d\n", cacheSeenCount())
	cd, _ := cacheDir()
	fmt.Printf("Taille totale du cache (~/.offpack): %s\n", formatSize(dirSize(cd)))
	if totalErrors > 0 {
		fmt.Printf("Erreurs: %d\n", totalErrors)
	}
}

func dirSize(path string) int64 {
	var size int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d o", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %so", float64(bytes)/float64(div), []string{"K", "M", "G", "T"}[exp])
}
