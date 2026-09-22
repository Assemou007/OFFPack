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
var cacheState = struct { sync.Mutex; seen map[string]bool }{seen: make(map[string]bool)}
var cacheDownloadSem = make(chan struct{}, 8)
func resetCacheSeen(){cacheState.Lock();cacheState.seen=make(map[string]bool);cacheState.Unlock()}
func markCacheSeen(key string) bool {cacheState.Lock();defer cacheState.Unlock();if cacheState.seen[key]{return false};cacheState.seen[key]=true;return true}
func withCacheNetwork(fn func() error) error {cacheDownloadSem<-struct{}{};defer func(){<-cacheDownloadSem}();return fn()}

func cacheDir() (string,error){home,err:=os.UserHomeDir();if err!=nil{return "",fmt.Errorf("impossible de trouver le répertoire home: %w",err)};dir:=filepath.Join(home,cacheDirName);if err=os.MkdirAll(dir,0755);err!=nil{return "",fmt.Errorf("impossible de créer %s: %w",dir,err)};return dir,nil}
func packageCacheDir(n,v string)(string,error){cd,e:=cacheDir();if e!=nil{return "",e};d:=filepath.Join(cd,sanitizeName(n),v);if e=os.MkdirAll(d,0755);e!=nil{return "",e};return d,nil}
func cachedTarballPath(n,v string)(string,error){d,e:=packageCacheDir(n,v);if e!=nil{return "",e};return filepath.Join(d,"package.tgz"),nil}
func isCached(n,v string)bool{p,e:=cachedTarballPath(n,v);if e!=nil{return false};_,e=os.Stat(p);return e==nil}
type CachedPackage struct{Name string `json:"name"`;Version string `json:"version"`;Dependencies map[string]string `json:"dependencies"`;Integrity string `json:"integrity,omitempty"`}
func saveCacheManifest(n,v string,deps map[string]string,integrity ...string)error{d,e:=packageCacheDir(n,v);if e!=nil{return e};m:=CachedPackage{Name:n,Version:v,Dependencies:deps};if len(integrity)>0{m.Integrity=integrity[0]};b,e:=json.MarshalIndent(m,"","  ");if e!=nil{return e};return os.WriteFile(filepath.Join(d,"manifest.json"),b,0644)}
func loadCacheManifest(n,v string)(*CachedPackage,error){d,e:=packageCacheDir(n,v);if e!=nil{return nil,e};b,e:=os.ReadFile(filepath.Join(d,"manifest.json"));if e!=nil{return nil,e};var m CachedPackage;e=json.Unmarshal(b,&m);return &m,e}
func resolveVersion(meta *PackageMeta,v string)string{if v=="latest"||v==""{return meta.DistTags.Latest};if _,ok:=meta.Versions[v];ok{return v};vs:=make([]string,0,len(meta.Versions));for x:=range meta.Versions{vs=append(vs,x)};return resolveBestVersion(vs,v)}
func unsanitizeName(n string)string{if strings.HasPrefix(n,"@"){if i:=strings.Index(n[1:],"_");i>=0{return n[:i+1]+"/"+n[i+2:]}};return n}
func sanitizeName(n string)string{var b strings.Builder;for _,r:=range n{if strings.ContainsRune(`/\\:*?"<>|`,r){b.WriteRune('_')}else{b.WriteRune(r)}};return b.String()}
func cacheKey(n,v string)string{return n+"@"+v}

func cachePackage(name,version string)error{meta,err:=withMeta(name);if err!=nil{return err};resolved:=resolveVersion(meta,version);if resolved==""{return fmt.Errorf("impossible de résoudre la version %s pour %s",version,name)};key:=cacheKey(name,resolved);if !markCacheSeen(key)||isCached(name,resolved){return nil};data,ok:=meta.Versions[resolved];if !ok{return fmt.Errorf("version %s introuvable pour %s",resolved,name)};if data.Dist.Tarball==""{return fmt.Errorf("aucun tarball pour %s@%s",name,resolved)};blob,err:=withCacheNetwork(func()error{var e error;blob,e=downloadTarball(data.Dist.Tarball,data.Dist.Integrity);return e});if err!=nil{return err};path,err:=cachedTarballPath(name,resolved);if err!=nil{return err};if err=os.WriteFile(path,blob,0644);err!=nil{return err};if err=saveCacheManifest(name,resolved,data.Dependencies,data.Dist.Integrity);err!=nil{return err};fmt.Printf("  OK %s@%s\n",name,resolved)
	var wg sync.WaitGroup; var mu sync.Mutex; var firstErr error; for dep, constraint:=range data.Dependencies {dep, constraint:=dep,constraint;wg.Add(1);go func(){defer wg.Done();if e:=cachePackage(dep,constraint);e!=nil{mu.Lock();if firstErr==nil{firstErr=e};mu.Unlock();fmt.Fprintf(os.Stderr,"  ATTENTION: dépendance %s non téléchargée: %v\n",dep,e)}}()};wg.Wait();return firstErr}
func withMeta(name string)(*PackageMeta,error){var m *PackageMeta;err:=withCacheNetwork(func()error{var e error;m,e=fetchPackageMeta(name);return e});return m,err}

func cmdCache(){pkgPath:="package.json";if len(os.Args)>2{pkgPath=os.Args[2]};pkg,err:=readPackageJSON(pkgPath);if err!=nil{fmt.Fprintf(os.Stderr,"Erreur: %v\n",err);os.Exit(1)};deps:=mergeDeps(pkg);if len(deps)==0{fmt.Println("Aucune dépendance trouvée dans package.json");return};resetCacheSeen();fmt.Printf("Téléchargement de %d dépendances dans le cache...\n",len(deps));cachePackages(deps)}
func mergeDeps(pkg *PackageJSON)map[string]string{d:=make(map[string]string);for n,v:=range pkg.Dependencies{d[n]=v};for n,v:=range pkg.DevDependencies{d[n]=v};return d}
func cachePackages(deps map[string]string){var wg sync.WaitGroup;var mu sync.Mutex;errors:=0;for n,v:=range deps{n,v:=n,v;wg.Add(1);go func(){defer wg.Done();if e:=cachePackage(n,v);e!=nil{fmt.Fprintf(os.Stderr,"  ÉCHEC %s: %v\n",n,e);mu.Lock();errors++;mu.Unlock()}}()};wg.Wait();if errors>0{fmt.Printf("\n%d erreur(s) rencontrée(s)\n",errors)}else{fmt.Println("\nToutes les dépendances sont en cache !")}}
