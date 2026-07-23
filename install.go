package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func cmdInstall() {
	pkgPath := "package.json"
	if len(os.Args) > 2 {
		pkgPath = os.Args[2]
	}

	pkg, err := readPackageJSON(pkgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		os.Exit(1)
	}

	allDeps := make(map[string]string)
	for name, version := range pkg.Dependencies {
		allDeps[name] = version
	}
	for name, version := range pkg.DevDependencies {
		allDeps[name] = version
	}

	if len(allDeps) == 0 {
		fmt.Println("Aucune dépendance trouvée dans package.json")
		return
	}

	projectDir := filepath.Dir(pkgPath)
	nodeModules := filepath.Join(projectDir, "node_modules")

	fmt.Printf("Installation de %d dépendances depuis le cache...\n", len(allDeps))
	installPackages(allDeps, nodeModules)
}

func installPackages(deps map[string]string, nodeModulesDir string) {
	errors := installAll(deps, nodeModulesDir)

	if errors > 0 {
		fmt.Printf("\n%d erreur(s) rencontrée(s)\n", errors)
	} else {
		fmt.Println("\nInstallation terminée avec succès !")
	}
}

func installAll(deps map[string]string, nodeModulesDir string) int {
	type result struct {
		name string
		err  error
	}
	results := make(chan result, len(deps))
	count := 0

	for name, version := range deps {
		count++
		go func(name, version string) {
			err := installPackage(name, version, nodeModulesDir)
			results <- result{name, err}
		}(name, version)
	}

	errors := 0
	for i := 0; i < count; i++ {
		r := <-results
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "  ÉCHEC %s: %v\n", r.name, r.err)
			errors++
		}
	}
	return errors
}

func installPackage(name, version, nodeModulesDir string) error {
	destDir := filepath.Join(nodeModulesDir, name)

	cachedDirs, err := findCachedVersions(name)
	if err != nil || len(cachedDirs) == 0 {
		return fmt.Errorf("%s n'est pas dans le cache. Lance d'abord 'offpack cache'", name)
	}

	resolvedVersion := resolveVersionFromCache(cachedDirs, version)
	if resolvedVersion == "" {
		return fmt.Errorf("aucune version de %s satisfaisant %s dans le cache", name, version)
	}

	tarballPath, _ := cachedTarballPath(name, resolvedVersion)
	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("impossible de nettoyer %s: %w", destDir, err)
	}

	if err := extractTarball(tarballPath, destDir); err != nil {
		return fmt.Errorf("erreur d'extraction de %s: %w", name, err)
	}

	fmt.Printf("  OK %s@%s\n", name, resolvedVersion)

	manifest, err := loadCacheManifest(name, resolvedVersion)
	if err == nil && len(manifest.Dependencies) > 0 {
		subNodeModules := filepath.Join(destDir, "node_modules")
		subDeps := make(map[string]string)
		for depName, depVer := range manifest.Dependencies {
			subDeps[depName] = depVer
		}
		if len(subDeps) > 0 {
			installAll(subDeps, subNodeModules)
		}
	}

	return nil
}

func resolveVersionFromCache(cachedVersions []string, constraint string) string {
	if constraint == "latest" || constraint == "" {
		return findLatestVersion(cachedVersions)
	}
	for _, cv := range cachedVersions {
		if cv == constraint {
			return cv
		}
	}
	return resolveBestVersion(cachedVersions, constraint)
}

func findCachedVersions(name string) ([]string, error) {
	cd, err := cacheDir()
	if err != nil {
		return nil, err
	}

	pkgCacheDir := filepath.Join(cd, sanitizeName(name))
	entries, err := os.ReadDir(pkgCacheDir)
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, e := range entries {
		if e.IsDir() {
			if _, ok := parseSemver(e.Name()); ok {
				versions = append(versions, e.Name())
			}
		}
	}
	return versions, nil
}
