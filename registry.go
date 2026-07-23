package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const npmRegistry = "https://registry.npmjs.org"

type PackageMeta struct {
	Name     string `json:"name"`
	DistTags struct {
		Latest string `json:"latest"`
	} `json:"dist-tags"`
	Versions map[string]struct {
		Name            string `json:"name"`
		Version         string `json:"version"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
		Dist            struct {
			Tarball string `json:"tarball"`
		} `json:"dist"`
	} `json:"versions"`
}

func fetchPackageMeta(name string) (*PackageMeta, error) {
	url := fmt.Sprintf("%s/%s", npmRegistry, strings.TrimSpace(name))
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erreur de connexion à %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("package %s introuvable (HTTP %d): %s", name, resp.StatusCode, string(body))
	}

	var meta PackageMeta
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("erreur de décodage JSON pour %s: %w", name, err)
	}

	return &meta, nil
}

func downloadTarball(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erreur de téléchargement %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("échec téléchargement %s (HTTP %d)", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erreur de lecture %s: %w", url, err)
	}

	return data, nil
}
