# offpack

**Gestionnaire de dépendances npm hors-ligne** — zéro dépendance, écrit en Go.

offpack télécharge une fois les packages npm et leurs dépendances dans un cache local, puis permet de les installer sans connexion internet. Idéal pour les environnements isolés, les CI restreints, ou simplement pour accélérer vos installations.

## Contexte

Dans les environnements sans accès à internet (serveurs air-gappés, zones de travail sécurisées, déploiements offline), `npm install` échoue. Les solutions existantes (npm proxy, registry miroir) sont lourdes à déployer.

offpack résout ce problème simplement : téléchargez les packages une fois sur une machine connectée, transférez le cache, et installez hors-ligne.

## Fonctionnalités

- **Cache local** dans `~/.offpack/` — téléchargement unique avec toutes les dépendances transitives
- **Installation offline** — `offpack install` lit le `package.json` et installe depuis le cache
- **Resolution semver** — résolution de versions intégrée (sans bibliothèque externe)
- **Mode populaire** — pré-télécharge les ~700 packages npm les plus utilisés et leurs dépendances
- **Mise à jour** — vérifie et télécharge les dernières versions disponibles
- **Zéro dépendance** — uniquement la stdlib Go (net/http, archive/tar, compress/gzip)

## Installation

```bash
git clone https://github.com/Assemou007/OFFPack.git
cd offpack
go build -o offpack .
./offpack self-install
```

**Prérequis** : Go 1.21+

La commande `self-install` détecte automatiquement votre OS :

| OS      | Destination                        |
|---------|------------------------------------|
| Linux   | `~/.local/bin/` (ou `/usr/local/bin` si accessible) |
| macOS   | `/usr/local/bin/`                  |
| Windows | `%USERPROFILE%\.local\bin\`        |

## Commandes

### `cache [chemin/vers/package.json]`

Télécharge les dépendances d'un `package.json` dans le cache local.

Lit les champs `dependencies` et `devDependencies`, puis télécharge chaque package et **toutes ses dépendances transitives** récursivement.

```bash
offpack cache                          # utilise package.json du dossier courant
offpack packages/front/package.json    # ou un chemin personnalisé
```

### `install [chemin/vers/package.json]`

Installe les dépendances depuis le cache local vers `node_modules/`. Ne fait **aucun appel réseau** — tout doit être dans `~/.offpack/` (lancez `cache` ou `fetch-popular` au préalable).

```bash
offpack install                          # installe dans ./node_modules/
offpack install chemin/vers/package.json # depuis un chemin personnalisé
```

Installe aussi les sous-dépendances dans `node_modules/<package>/node_modules/`.

### `fetch-popular [nombre]`

Télécharge les packages npm les plus populaires (liste hardcodée d'environ 700 packages) et leurs dépendances dans le cache. Idéal pour pré-remplir le cache avant d'aller hors-ligne.

```bash
offpack fetch-popular       # 500 premiers (défaut)
offpack fetch-popular 700   # tous
offpack fetch-popular 10    # juste quelques-uns pour tester
```

### `get <package>[@version] [--dev|-D]`

Télécharge depuis npm, installe dans `node_modules/`, et ajoute au `package.json` du projet courant.

```bash
offpack get react                        # dernière version
offpack get lodash@4.17.21               # version spécifique
offpack get typescript --dev             # en devDependencies
offpack get @angular/core                # package scoped (un seul à la fois)
```

La version installée est écrite dans `package.json` avec un caret (`^4.17.21`).

### `update`

Parcourt tous les packages dans `~/.offpack/`, vérifie la dernière version disponible sur npm, et télécharge les nouvelles versions dans le cache.

```bash
offpack update
```

Affiche un compteur, les mises à jour trouvées, et un résumé final (temps, paquets mis à jour, erreurs, taille du cache).

### `self-install`

Copie le binaire dans le dossier système approprié selon votre OS (nécessite les droits d'écriture). Pratique après une compilation.

```bash
./offpack self-install
```

| OS      | Destination                        |
|---------|------------------------------------|
| Linux   | `~/.local/bin/` (ou `/usr/local/bin` si accessible) |
| macOS   | `/usr/local/bin/`                  |
| Windows | `%USERPROFILE%\.local\bin\`        |

### `help`

Affiche la liste des commandes et leur usage.

## Architecture

```
~/.offpack/
├── <package>/
│   └── <version>/
│       ├── package.tgz     # tarball du package
│       └── manifest.json   # métadonnées + dépendances
└── ...
```

Le projet est structuré en 11 fichiers Go (~1500 lignes), sans dépendance externe :

| Fichier | Rôle |
|---------|------|
| `main.go` | Dispatch CLI |
| `registry.go` | Appels HTTP à `registry.npmjs.org` |
| `cache.go` | Gestion du cache local |
| `resolver.go` | Résolution semver |
| `extract.go` | Extraction des tarballs `.tgz` |
| `install.go` | Installation depuis le cache |
| `get.go` | Installation + mise à jour de `package.json` |
| `popular.go` | Liste des packages populaires |
| `update.go` | Mise à jour du cache |
| `packagejson.go` | Parseur/écriture de `package.json` |
| `selfinstall.go` | Auto-installation multi-plateforme |

## Licence

MIT
