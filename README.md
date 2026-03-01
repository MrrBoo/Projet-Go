# FileOps / WebOps - Outil d'analyse de fichiers en Go

Ce projet est un outil en ligne de commande développé en Go qui permet d'analyser des fichiers texte (stats, filtres, fusion, rapports) et de récupérer du contenu depuis Wikipédia pour lui appliquer les mêmes traitements.

---

## Prérequis avant de lancer le script

- Initialiser le module Go du projet avec `go mod init <nom_du_module>`, ce qui génère le fichier go.mod nécessaire à la gestion des dépendances.
- Installer goquery avec `go get github.com/PuerkitoBio/goquery`, la bibliothèque utilisée pour parser le HTML des pages Wikipédia.

---

## Lancement

```bash
go run main.go
```

---

## Configuration - `config.txt`

Le fichier config.txt centralise les paramètres du programme (fichier par défaut, dossiers d'entrée et de sortie) et est lu automatiquement au démarrage.


```
default_file=data/input.txt
base_dir=data
out_dir=out
default_ext=.txt
```

Les lignes vides et les lignes commençant par `#` sont ignorées. Si une clé est absente, une valeur par défaut est utilisée.

---

## Fonctionnalités implémentées

### Niveau 10/20 — FileOps

**Choix 1 - Changer le fichier courant**  
Affiche les fichiers disponibles dans `base_dir` et permet d'en choisir un. Si aucun fichier n'est saisi, le fichier par défaut de la config est conservé.

**Choix 2 - Analyse d'un fichier (Choix A)**  
Prend le fichier courant et applique en une seule fois :
- Affichage des métadonnées : taille en octets, date de modification, nombre de lignes
- Statistiques sur les mots : nombre total (les tokens purement numériques sont exclus) et longueur moyenne
- Comptage des lignes contenant un mot-clé saisi par l'utilisateur
- Génération de `out/filtered.txt` — lignes contenant le mot-clé
- Génération de `out/filtered_not.txt` — lignes ne contenant pas le mot-clé
- Génération de `out/head.txt` — N premières lignes
- Génération de `out/tail.txt` — N dernières lignes

**Choix 3 - Analyse multi-fichiers (Choix B)**  
Prend un répertoire (ou `base_dir` par défaut) et traite tous les `.txt` qu'il contient :
- Calcul des stats pour chaque fichier (taille, lignes, mots, longueur moyenne)
- Génération de `out/report.txt` — rapport global lisible avec totaux
- Génération de `out/index.txt` — tableau chemin / taille / date de modification
- Génération de `out/merged.txt` — tous les fichiers concaténés avec séparateurs

### Niveau 12/20 - WebOps (Wikipédia)

**Choix 4 - Analyse d'un article Wikipédia (Choix C)**  
L'utilisateur saisit le nom d'un article (ex: `Go_(langage)`). Le programme :
- Télécharge la page `https://fr.wikipedia.org/wiki/<article>` via `net/http`
- Extrait le texte de tous les paragraphes avec `goquery` (pas de regex)
- Affiche les statistiques mots (réutilisation de `wordStats`)
- Filtre les paragraphes contenant un mot-clé (réutilisation de `filterLines`)
- Écrit le résultat dans `out/wiki_<article>.txt`

---

## Description du travail effectué

**Démarrage et configuration**

Au lancement, `main()` appelle `LoadConfig()` qui ouvre `config.txt` ligne par ligne via un `bufio.Scanner` — un lecteur bufferisé qui lit le fichier sans tout charger en mémoire d'un coup. Chaque ligne est découpée sur le `=` pour en extraire une clé et une valeur, qui viennent remplir une struct `Config`. Les lignes vides ou commentées sont ignorées. Si une clé est absente du fichier, la valeur par défaut définie dans `DefaultConfig()` est conservée. La struct `Config` est ensuite passée à toutes les fonctions du programme, ce qui centralise l'accès aux chemins et paramètres.

**Menu interactif**

`showMenu()` tourne en boucle et utilise un `bufio.Reader` pour lire ce que l'utilisateur tape dans le terminal — ce reader lit le flux d'entrée standard (`os.Stdin`) caractère par caractère jusqu'au retour à la ligne. La saisie passe par `readInt()` qui convertit l'entrée en entier, puis un `switch` sur ce nombre dispatche vers la bonne fonctionnalité. La boucle ne s'arrête que si l'utilisateur choisit l'option "Quitter".

**Choix 2 — Analyse d'un fichier (`runChoiceA`)**

Le fichier courant est lu en entier via `readAllLines()` qui utilise un `bufio.Scanner` pour récupérer chaque ligne dans un slice de strings. Ce slice est ensuite passé à plusieurs fonctions :

- `printFileInfos()` appelle `os.Stat()` qui interroge le système de fichiers pour récupérer les métadonnées du fichier (taille, date de modification)
- `wordStats()` parcourt chaque ligne, découpe les mots avec `strings.Fields()` qui sépare sur les espaces et tabulations, exclut les tokens purement numériques via `isNumericWord()`, et calcule le total et la longueur moyenne
- `countLinesWithKeyword()` et `filterLines()` utilisent `strings.Contains()` pour chercher le mot-clé dans chaque ligne
- `head()` et `tail()` retournent simplement les N premiers ou derniers éléments du slice
- Chaque résultat est écrit dans un fichier de sortie via `writeLines()` qui utilise un `bufio.Writer` — l'équivalent du Scanner mais en écriture, qui accumule les données en mémoire avant de les écrire d'un seul coup sur le disque pour de meilleures performances

**Choix 3 — Analyse multi-fichiers (`runChoiceB`)**

`listFilesWithExt()` parcourt le répertoire avec `os.ReadDir()` qui retourne la liste des entrées du dossier, et filtre les fichiers selon leur extension. Pour chaque fichier trouvé, `analyzeOneFile()` appelle `os.Stat()` et `wordStats()` et stocke le résultat dans une struct `FileSummary` — une structure qui regroupe toutes les infos d'un fichier en un seul objet. Une fois tous les fichiers analysés, trois fichiers de sortie sont générés : `writeReport()` produit un rapport lisible avec des totaux agrégés, `writeIndex()` génère un tableau tabulé chemin/taille/date, et `mergeFiles()` concatène le contenu de tous les fichiers avec des séparateurs.

**Choix 4 — Wikipédia (`runChoiceC`)**

Une requête HTTP GET est construite avec `http.NewRequest()` et envoyée via un `http.Client` — le client HTTP natif de Go qui gère la connexion, les headers et la réponse. Le corps de la réponse est passé directement à `goquery.NewDocumentFromReader()` qui parse le HTML et construit un arbre DOM navigable. Le sélecteur `#mw-content-text p` cible uniquement les paragraphes du contenu principal de la page, dont le texte est extrait avec `.Text()` qui supprime toutes les balises HTML pour ne garder que le contenu brut, et stocké dans un slice de strings. Ce slice est ensuite traité avec les mêmes fonctions que pour un fichier local (`wordStats`, `filterLines`), ce qui évite toute duplication de logique entre l'analyse fichier et l'analyse web.
