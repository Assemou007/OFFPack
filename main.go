package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "cache":
		cmdCache()
	case "install":
		cmdInstall()
	case "fetch-popular":
		cmdFetchPopular()
	case "update":
		cmdUpdate()
	case "get":
		cmdGet()
	case "self-install":
		cmdSelfInstall()
	case "help", "--help", "-h", "-help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Commande inconnue: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`offpack - Gestionnaire de dépendances hors-ligne

Commandes:
  cache     Télécharge les dépendances d'un projet dans le cache local
            Usage: offpack cache [chemin/vers/package.json]

  install   Installe les dépendances depuis le cache local
            Usage: offpack install [chemin/vers/package.json]

  fetch-popular  Télécharge les packages les plus populaires du registry npm
            dans le cache local (pour usage hors-ligne)
            Usage: offpack fetch-popular [nombre]
            Exemple: offpack fetch-popular 500 (défaut: 500, max: 1000)

  get       Télécharge et installe un package spécifique dans node_modules
            Usage: offpack get <package>[@version]
            Exemple: offpack get express, offpack get lodash@4.18.1

  update    Vérifie et télécharge les dernières versions des packages
            dans le cache (met à jour vers les versions récentes)
            Usage: offpack update

  self-install  Installe le binaire offpack dans le PATH selon l'OS
            Usage: offpack self-install

  help      Affiche cette aide`)
}
