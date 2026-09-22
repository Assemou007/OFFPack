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
	cwd, _ := os.Getwd()
	node := filepath.Join(cwd, "node_modules")
	resetCacheSeen()
	if err := cachePackage(name, version); err != nil {
		fmt.Fprintf(os.Stderr, "ÉCHEC: %v\n", err)
		os.Exit(1)
	}
	if err := installPackage(name, version, node); err != nil {
		fmt.Fprintf(os.Stderr, "ÉCHEC installation: %v\n", err)
		os.Exit(1)
	}
	actual := readInstalledVersion(node, name)
	if actual != "" {
		if err := setPackageJSONDependency(filepath.Join(cwd, "package.json"), name, actual, isDev); err != nil {
			fmt.Fprintf(os.Stderr, "ATTENTION: package.json non mis à jour: %v\n", err)
		} else {
			fmt.Printf("✅ %s@%s ajouté à package.json\n", name, actual)
		}
	}
	fmt.Printf("✅ %s installé dans node_modules/\n", name)
}
func readInstalledVersion(node, name string) string {
	data, err := os.ReadFile(filepath.Join(node, name, "package.json"))
	if err != nil {
		return ""
	}
	var p struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(data, &p) != nil {
		return ""
	}
	return p.Version
}
func setPackageJSONDependency(path, name, version string, dev bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("impossible de lire package.json: %w", err)
	}
	var doc map[string]json.RawMessage
	if err = json.Unmarshal(data, &doc); err != nil {
		return err
	}
	field := "dependencies"
	other := "devDependencies"
	if dev {
		field, other = other, field
	}
	var deps map[string]string
	_ = json.Unmarshal(doc[field], &deps)
	if deps == nil {
		deps = map[string]string{}
	}
	var otherDeps map[string]string
	_ = json.Unmarshal(doc[other], &otherDeps)
	delete(otherDeps, name)
	deps[name] = version
	a, _ := json.Marshal(deps)
	b, _ := json.Marshal(otherDeps)
	doc[field] = a
	doc[other] = b
	out, _ := json.MarshalIndent(doc, "", "  ")
	return os.WriteFile(path, append(out, '\n'), 0644)
}
func parsePackageSpec(spec string) (string, string) {
	if i := strings.LastIndex(spec, "@"); i > 0 {
		return spec[:i], spec[i+1:]
	}
	return spec, ""
}
