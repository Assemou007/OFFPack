package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func cmdGet() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: offpack get <package>[@version]")
		fmt.Fprintln(os.Stderr, "Exemple: offpack get express")
		fmt.Fprintln(os.Stderr, "         offpack get lodash@4.18.1")
		os.Exit(1)
	}

	target := os.Args[2]
	name, version := parsePackageSpec(target)
	if version == "" {
		version = "latest"
	}
	isDev := false
	for _, a := range os.Args[3:] {
		if a == "--dev" || a == "-D" {
			isDev = true
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		os.Exit(1)
	}
	nodeModules := filepath.Join(cwd, "node_modules")

	cacheSeen = make(map[string]bool)

	fmt.Printf("Installation de %s@%s...\n", name, version)

	err = cachePackage(name, version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ÉCHEC: %v\n", err)
		os.Exit(1)
	}

	err = installPackage(name, version, nodeModules)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ÉCHEC installation: %v\n", err)
		os.Exit(1)
	}

	actualVersion := readInstalledVersion(nodeModules, name)
	if actualVersion != "" {
		err = addToPackageJSON(cwd, name, actualVersion, isDev)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ATTENTION: package.json non mis à jour: %v\n", err)
		} else {
			fmt.Printf("✅ %s@%s ajouté à package.json\n", name, actualVersion)
		}
	}

	fmt.Printf("✅ %s installé dans node_modules/\n", name)
}

func readInstalledVersion(nodeModules, name string) string {
	pkgPath := filepath.Join(nodeModules, name, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return ""
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return ""
	}
	return "^" + pkg.Version
}

func addToPackageJSON(cwd, name, version string, dev bool) error {
	pkgPath := filepath.Join(cwd, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return fmt.Errorf("impossible de lire package.json: %w", err)
	}

	var pkg struct {
		Name            string            `json:"name,omitempty"`
		Version         string            `json:"version,omitempty"`
		Dependencies    map[string]string `json:"dependencies,omitempty"`
		DevDependencies map[string]string `json:"devDependencies,omitempty"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return fmt.Errorf("package.json invalide: %w", err)
	}

	if pkg.Dependencies == nil {
		pkg.Dependencies = make(map[string]string)
	}
	if pkg.DevDependencies == nil {
		pkg.DevDependencies = make(map[string]string)
	}

	if dev {
		delete(pkg.Dependencies, name)
		pkg.DevDependencies[name] = version
	} else {
		delete(pkg.DevDependencies, name)
		pkg.Dependencies[name] = version
	}

	cleaned, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(pkgPath, append(cleaned, '\n'), 0644)
}

func parsePackageSpec(spec string) (name, version string) {
	if idx := strings.LastIndex(spec, "@"); idx > 0 {
		return spec[:idx], spec[idx+1:]
	}
	return spec, ""
}
