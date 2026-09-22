package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type PackageJSON struct { Name string `json:"name"`; Version string `json:"version"`; Dependencies map[string]string `json:"dependencies"`; DevDependencies map[string]string `json:"devDependencies"` }
func readPackageJSON(path string)(*PackageJSON,error){data,err:=os.ReadFile(path);if err!=nil{return nil,fmt.Errorf("impossible de lire %s: %w",path,err)};var pkg PackageJSON;if err=json.Unmarshal(data,&pkg);err!=nil{return nil,fmt.Errorf("format JSON invalide dans %s: %w",path,err)};return &pkg,nil}

func updateResolvedPackageJSON(path string, resolved map[string]string) error { data,err:=os.ReadFile(path);if err!=nil{return err};var doc map[string]json.RawMessage;if err=json.Unmarshal(data,&doc);err!=nil{return err};for _,field:=range []string{"dependencies","devDependencies"}{var deps map[string]string;if raw,ok:=doc[field];ok{if json.Unmarshal(raw,&deps)!=nil{continue}};if deps==nil{continue};changed:=false;for name:=range deps{if v,ok:=resolved[name];ok{deps[name]=v;changed=true}};if changed{raw,_:=json.Marshal(deps);doc[field]=raw}};out,err:=json.MarshalIndent(doc,"","  ");if err!=nil{return err};return os.WriteFile(path,append(out,'\n'),0644)}
