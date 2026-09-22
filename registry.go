package main

import (
	"crypto/sha1"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const npmRegistry = "https://registry.npmjs.org"

type PackageMeta struct { Name string `json:"name"`; DistTags struct { Latest string `json:"latest"` } `json:"dist-tags"`; Versions map[string]struct { Name string `json:"name"`; Version string `json:"version"`; Dependencies map[string]string `json:"dependencies"`; DevDependencies map[string]string `json:"devDependencies"`; Dist struct { Tarball string `json:"tarball"`; Integrity string `json:"integrity"` } `json:"dist"` } `json:"versions"` }

func fetchPackageMeta(name string) (*PackageMeta, error) { url := fmt.Sprintf("%s/%s", npmRegistry, strings.TrimSpace(name)); resp, err := http.Get(url); if err != nil { return nil, fmt.Errorf("erreur de connexion à %s: %w", url, err) }; defer resp.Body.Close(); if resp.StatusCode != http.StatusOK { body,_:=io.ReadAll(resp.Body); return nil, fmt.Errorf("package %s introuvable (HTTP %d): %s", name, resp.StatusCode, body) }; var meta PackageMeta; if err:=json.NewDecoder(resp.Body).Decode(&meta); err!=nil{return nil,fmt.Errorf("erreur de décodage JSON pour %s: %w",name,err)}; return &meta,nil }

func downloadTarball(url, integrity string) ([]byte, error) { resp, err := http.Get(url); if err != nil { return nil, fmt.Errorf("erreur de téléchargement %s: %w", url, err) }; defer resp.Body.Close(); if resp.StatusCode != http.StatusOK { return nil, fmt.Errorf("échec téléchargement %s (HTTP %d)", url, resp.StatusCode) }; data, err := io.ReadAll(resp.Body); if err != nil { return nil, fmt.Errorf("erreur de lecture %s: %w",url,err) }; if integrity != "" { if err:=verifyIntegrity(data, integrity); err!=nil{return nil,err} }; return data,nil }

func verifyIntegrity(data []byte, integrity string) error { parts:=strings.SplitN(integrity,"-",2); if len(parts)!=2{return fmt.Errorf("intégrité npm invalide: %s",integrity)}; expected,err:=base64.StdEncoding.DecodeString(parts[1]);if err!=nil{return fmt.Errorf("empreinte npm invalide: %w",err)}; var actual []byte; switch parts[0] {case "sha512": sum:=sha512.Sum512(data);actual=sum[:];case "sha1":sum:=sha1.Sum(data);actual=sum[:];default:return fmt.Errorf("algorithme d'intégrité npm non supporté: %s",parts[0])}; if len(actual)!=len(expected)||subtle.ConstantTimeCompare(actual,expected)!=1{return fmt.Errorf("échec de vérification d'intégrité du tarball")};return nil }
