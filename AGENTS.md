# offpack

Single `package main` (~1500 loc), Go stdlib only, no deps, no tests, no linter config.

## Build & install

```bash
go build -o offpack .           # binary in .gitignore
go vet ./...                    # only lint available
./offpack self-install          # auto-detect OS, copy to PATH
```

## CLI dispatch

`main.go` switches on `os.Args[1]` — 7 commands: `cache`, `install`, `fetch-popular`, `get`, `update`, `self-install`, `help`. No version subcommand exists.

`selfinstall.go` uses `runtime.GOOS` to pick the install dir: Linux → `~/.local/bin/` (or `/usr/local/bin`), macOS → `/usr/local/bin`, Windows → `%USERPROFILE%\.local\bin\`.

`get.go` `parsePackageSpec` splits on **last** `@` — a package name like `@scope/pkg@^1.0` would parse incorrectly.

## Key pitfalls

- **`sanitizeName`** (`/` → `_` for scoped packages). Code reading cache dirs from `~/.offpack/` and calling the registry must `unsanitizeName` first (replaces first `_` after `@` with `/`). Defined in `cache.go`.
- **`cacheSeen`** (global in `cache.go`) — reset per command (`make(map[string]bool)`), no concurrent protection.
- **`cachePackage()` recurses** into all transitive dependencies — can be deep/slow.
- **`resolveBestVersion`** incomplete: no `*`, bare `"1"`, `"npm:xxx@^y"`, `v`-prefix in ranges.
- **`package.json` rewrite** (`json.MarshalIndent`) destroys original formatting.
- **Cache is global** (`~/.offpack/`) — no project isolation.
- **`topPackages`** (popular.go) is hardcoded ~700 entries — needs periodic refresh.
- **French** error/output messages throughout.
- **Module name** is `offpack` (not github.com/...).
- **No tests exist** — blind spots in semver resolver and sanitize/unsanitize roundtrip.
