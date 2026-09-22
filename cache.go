package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const cacheDirName = ".offpack"

var cacheState = struct {
	sync.Mutex
	seen map[string]bool
}{seen: make(map[string]bool)}

var cacheDownloadSem = make(chan struct{}, 8)

func resetCacheSeen() {
	cacheState.Lock()
	cacheState.seen = make(map[string]bool)
	cacheState.Unlock()
}

func markCacheSeen(key string) bool {
	cacheState.Lock()
	defer cacheState.Unlock()
	if cacheState.seen[key] {
		return false
	}
	cacheState.seen[key] = true
	return true
}

func cacheSeenCount() int {
	cacheState.Lock()
	defer cacheState.Unlock()
	return len(cacheState.seen)
}

func withCacheNetwork(fn func() error) error {
	cacheDownloadSem <- struct{}{}
	defer func() { <-cacheDownloadSem }()
	return fn()
}

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

func packageCacheDir(name, version string) (string, error) {
	cd, err := cacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cd, sanitizeName(name), version)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func cachedTarballPath(name, version string) (string, error) {
	dir, err := packageCacheDir(name, version)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "package.tgz"), nil
}

func isCached(name, version string) bool {
	path, err := cachedTarballPath(name, version)
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
	Integrity    string            `json:"integrity,omitempty"`
}

func saveCacheManifest(name, version string, deps map[string]string, integrity ...string) error {
	dir, err := packageCacheDir(name, version)
	if err != nil {
		return err
	}
	manifest := CachedPackage{Name: name, Version: version, Dependencies: deps}
	if len(integrity) > 0 {
		manifest.Integrity = integrity[0]
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0644)
}

func loadCacheManifest(name, version string) (*CachedPackage, error) {
	dir, err := packageCacheDir(name, version)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, err
	}
	var manifest CachedPackage
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func resolveVersion(meta *PackageMeta, version string) string {
	if version == "latest" || version == "" {
		return meta.DistTags.Latest
	}
	if _, ok := meta.Versions[version]; ok {
		return version
	}
	versions := make([]string, 0, len(meta.Versions))
	for candidate := range meta.Versions {
		versions = append(versions, candidate)
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
	var result strings.Builder
	for _, r := range name {
		if strings.ContainsRune(`/\\:*?"<>|`, r) {
			result.WriteRune('_')
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func cacheKey(name, version string) string { return name + "@" + version }

func cachePackage(name, version string) error {
	meta, err := withMeta(name)
	if err != nil {
		return err
	}
	resolved := resolveVersion(meta, version)
	if resolved == "" {
		return fmt.Errorf("impossible de résoudre la version %s pour %s", version, name)
	}
	if !markCacheSeen(cacheKey(name, resolved)) || isCached(name, resolved) {
		return nil
	}

	versionData, ok := meta.Versions[resolved]
	if !ok {
		return fmt.Errorf("version %s introuvable pour %s", resolved, name)
	}
	if versionData.Dist.Tarball == "" {
		return fmt.Errorf("aucun tarball pour %s@%s", name, resolved)
	}

	var blob []byte
	err = withCacheNetwork(func() error {
		var downloadErr error
		blob, downloadErr = downloadTarball(versionData.Dist.Tarball, versionData.Dist.Integrity)
		return downloadErr
	})
	if err != nil {
		return err
	}

	path, err := cachedTarballPath(name, resolved)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, blob, 0644); err != nil {
		return err
	}
	if err := saveCacheManifest(name, resolved, versionData.Dependencies, versionData.Dist.Integrity); err != nil {
		return err
	}
	fmt.Printf("  OK %s@%s\n", name, resolved)

	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex
	for dep, constraint := range versionData.Dependencies {
		dep, constraint := dep, constraint
		wg.Add(1)
		go func() {
			defer wg.Done()
			if depErr := cachePackage(dep, constraint); depErr != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = depErr
				}
				errMu.Unlock()
				fmt.Fprintf(os.Stderr, "  ATTENTION: dépendance %s non téléchargée: %v\n", dep, depErr)
			}
		}()
	}
	wg.Wait()
	return firstErr
}

func withMeta(name string) (*PackageMeta, error) {
	var meta *PackageMeta
	err := withCacheNetwork(func() error {
		var fetchErr error
		meta, fetchErr = fetchPackageMeta(name)
		return fetchErr
	})
	return meta, err
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
	deps := mergeDeps(pkg)
	if len(deps) == 0 {
		fmt.Println("Aucune dépendance trouvée dans package.json")
		return
	}
	resetCacheSeen()
	fmt.Printf("Téléchargement de %d dépendances dans le cache...\n", len(deps))
	cachePackages(deps)
}

func mergeDeps(pkg *PackageJSON) map[string]string {
	deps := make(map[string]string)
	for name, version := range pkg.Dependencies {
		deps[name] = version
	}
	for name, version := range pkg.DevDependencies {
		deps[name] = version
	}
	return deps
}

func cachePackages(deps map[string]string) {
	var wg sync.WaitGroup
	var errMu sync.Mutex
	errors := 0
	for name, version := range deps {
		name, version := name, version
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := cachePackage(name, version); err != nil {
				fmt.Fprintf(os.Stderr, "  ÉCHEC %s: %v\n", name, err)
				errMu.Lock()
				errors++
				errMu.Unlock()
			}
		}()
	}
	wg.Wait()
	if errors > 0 {
		fmt.Printf("\n%d erreur(s) rencontrée(s)\n", errors)
	} else {
		fmt.Println("\nToutes les dépendances sont en cache !")
	}
}
