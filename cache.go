package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const cacheDirName = ".offpack"

func cacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("impossible de trouver le répertoire home: %w", err)
	}
	dir := filepath.Join(home, cacheDirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("impossible de créer %s: %w", dir, err)
	}
	return dir, nil
}

func packageCacheDir(pkgName, version string) (string, error) {
	cd, err := cacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cd, sanitizeName(pkgName), version)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("impossible de créer %s: %w", dir, err)
	}
	return dir, nil
}

func cachedTarballPath(pkgName, version string) (string, error) {
	dir, err := packageCacheDir(pkgName, version)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "package.tgz"), nil
}

func isCached(pkgName, version string) bool {
	path, err := cachedTarballPath(pkgName, version)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

type CachedPackage struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
}

func saveCacheManifest(pkgName, version string, deps map[string]string) error {
	dir, err := packageCacheDir(pkgName, version)
	if err != nil {
		return err
	}
	manifest := CachedPackage{
		Name:         pkgName,
		Version:      version,
		Dependencies: deps,
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0644)
}

func loadCacheManifest(pkgName, version string) (*CachedPackage, error) {
	dir, err := packageCacheDir(pkgName, version)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, err
	}
	var m CachedPackage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func resolveVersion(meta *PackageMeta, version string) string {
	if version == "latest" || version == "" {
		return meta.DistTags.Latest
	}

	if _, ok := meta.Versions[version]; ok {
		return version
	}

	var versions []string
	for v := range meta.Versions {
		versions = append(versions, v)
	}

	return resolveBestVersion(versions, version)
}

func unsanitizeName(name string) string {
	if strings.HasPrefix(name, "@") {
		if idx := strings.Index(name[1:], "_"); idx >= 0 {
			return name[:idx+1] + "/" + name[idx+2:]
		}
	}
	return name
}

func sanitizeName(name string) string {
	var result []rune
	for _, r := range name {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			result = append(result, '_')
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func cmdCache() {
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

	fmt.Printf("Téléchargement de %d dépendances dans le cache...\n", len(allDeps))
	cachePackages(allDeps)
}

func cachePackages(deps map[string]string) {
	type result struct {
		name string
		err  error
	}
	results := make(chan result, len(deps))
	count := 0

	for name, version := range deps {
		count++
		go func(name, version string) {
			err := cachePackage(name, version)
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

	if errors > 0 {
		fmt.Printf("\n%d erreur(s) rencontrée(s)\n", errors)
	} else {
		fmt.Println("\nToutes les dépendances sont en cache !")
	}
}

var cacheSeen = make(map[string]bool)

func cacheKey(name, version string) string {
	return name + "@" + version
}

func cachePackage(name, version string) error {
	meta, err := fetchPackageMeta(name)
	if err != nil {
		return err
	}

	resolvedVersion := resolveVersion(meta, version)
	if resolvedVersion == "" {
		return fmt.Errorf("impossible de résoudre la version %s pour %s", version, name)
	}

	key := cacheKey(name, resolvedVersion)
	if cacheSeen[key] {
		return nil
	}
	cacheSeen[key] = true

	if isCached(name, resolvedVersion) {
		return nil
	}

	versionData, ok := meta.Versions[resolvedVersion]
	if !ok {
		return fmt.Errorf("version %s introuvable pour %s", resolvedVersion, name)
	}

	tarballURL := versionData.Dist.Tarball
	if tarballURL == "" {
		return fmt.Errorf("aucun tarball pour %s@%s", name, resolvedVersion)
	}

	data, err := downloadTarball(tarballURL)
	if err != nil {
		return err
	}

	path, err := cachedTarballPath(name, resolvedVersion)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("impossible d'écrire %s: %w", path, err)
	}

	if err := saveCacheManifest(name, resolvedVersion, versionData.Dependencies); err != nil {
		return fmt.Errorf("impossible de sauvegarder le manifest: %w", err)
	}

	fmt.Printf("  OK %s@%s\n", name, resolvedVersion)

	for depName, depConstraint := range versionData.Dependencies {
		if err := cachePackage(depName, depConstraint); err != nil {
			fmt.Fprintf(os.Stderr, "  ATTENTION: dépendance %s non téléchargée: %v\n", depName, err)
		}
	}

	return nil
}
