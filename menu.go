// Projet de traitement de fichiers texte en Go
// Stanislas DE DIEULEVEULT M1 DOA

package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
)

type Config struct {
	DefaultFile string
	BaseDir     string
	OutDir      string
	DefaultExt  string
}

type FileSummary struct {
	Path      string
	SizeBytes int64
	ModTime   string
	Lines     int
	Words     int
	AvgWordLn float64
	Err       error
}

const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
)

func DefaultConfig() Config {
	return Config{
		DefaultFile: "data/test.txt",
		BaseDir:     "data",
		OutDir:      "out",
		DefaultExt:  ".txt",
	}
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()

	file, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "default_file":
			cfg.DefaultFile = value
		case "base_dir":
			cfg.BaseDir = value
		case "out_dir":
			cfg.OutDir = value
		case "default_ext":
			cfg.DefaultExt = value
		}
	}

	if err := scanner.Err(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func main() {
	cfg, err := LoadConfig("config.txt")
	if err != nil {
		fmt.Println("Erreur : fichier config.txt introuvable ou illisible")
		return
	}

	if err := os.MkdirAll(cfg.OutDir, 0755); err != nil {
		fmt.Println("Erreur : impossible de créer le dossier out/")
		return
	}

	reader := bufio.NewReader(os.Stdin)

	currentFile := cfg.DefaultFile
	if !isValidFile(currentFile) {
		fmt.Println("Erreur : default_file invalide (n'existe pas ou pas un fichier)", currentFile)
		return
	}

	showMenu(reader, cfg, &currentFile)
}

func showMenu(reader *bufio.Reader, cfg Config, currentFile *string) {
	for {
		fmt.Println("Menu :")
		fmt.Println(Cyan + "1. Choisir le fichier" + Reset)
		fmt.Println(Cyan + "2. Analyse du fichier" + Reset)
		fmt.Println(Cyan + "3. Analyse multifichiers" + Reset)
		fmt.Println(Cyan + "4. Analyse Wikipédia" + Reset)
		fmt.Println(Cyan + "5. Quitter" + Reset)
		fmt.Println("\nFichier actuel :", Yellow, *currentFile, Reset)

		choice := readInt(reader, "\nChoisissez une option du menu : ")

		switch choice {
		case 1:
			*currentFile = chooseFile(reader, cfg)
			fmt.Println("Fichier courant mis à jour :", *currentFile)

		case 2:
			runChoiceA(reader, cfg, *currentFile)

		case 3:
			runChoiceB(reader, cfg)
			fmt.Println("Analyse multifichiers")
		case 4:
			runChoiceC(reader, cfg)
		case 5:
			fmt.Println("Quitter")
			return
		default:
			fmt.Println(Red + "Veuillez choisir une option valide." + Reset)
		}
	}
}

func chooseFile(reader *bufio.Reader, cfg Config) string {
	fmt.Println("Fichiers disponibles dans", cfg.BaseDir, ":")

	entries, err := os.ReadDir(cfg.BaseDir)
	if err != nil {
		fmt.Println("WARN: impossible de lire", cfg.BaseDir, "Le fichier par défaut sera utilisé")
		return cfg.DefaultFile
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), strings.ToLower(cfg.DefaultExt)) {
			fmt.Println("-", e.Name())
		}
	}

	input := readLine(reader, "Nom du fichier à analyser (si aucun fichier choisi, le fichier par défaut sera utilisé) : ")

	if input == "" {
		return cfg.DefaultFile
	}

	path := filepath.Join(cfg.BaseDir, input)
	if isValidFile(path) {
		return path
	}

	fmt.Println(Red+"Fichier invalide, utilisation du fichier par défaut :"+Reset, cfg.DefaultFile)
	return cfg.DefaultFile
}

func isValidFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func runChoiceA(reader *bufio.Reader, cfg Config, filePath string) {
	if !isValidFile(filePath) {
		fmt.Println(Red+"Erreur : fichier invalide"+Reset, filePath)
		return
	}

	keyword := readLine(reader, "Mot-clé à chercher : ")
	n := readInt(reader, "Nombre de lignes pour head/tail : ")

	lines, err := readAllLines(filePath)
	if err != nil {
		fmt.Println(Red+"Erreur lecture fichier :"+Reset, err)
		return
	}

	if err := printFileInfos(filePath, len(lines)); err != nil {
		fmt.Println(Red+"Erreur infos fichier :"+Reset, err)
		return
	}

	totalWords, avgLen := wordStats(lines)
	fmt.Println("\nStats mots :")
	fmt.Println("Nombre de mots :", Green, totalWords, Reset)
	fmt.Printf("Longueur moyenne : %s%.2f%s\n", Green, avgLen, Reset)

	count := countLinesWithKeyword(lines, keyword)
	fmt.Println("Nombre de lignes contenant le mot", Green, strconv.Quote(keyword), Reset, ":", Green, count, Reset)

	filtered := filterLines(lines, keyword, true)
	if err := writeLines(filepath.Join(cfg.OutDir, "filtered.txt"), filtered); err != nil {
		fmt.Println(Red+"Erreur écriture filtered.txt :"+Reset, err)
		return
	}

	filteredNot := filterLines(lines, keyword, false)
	if err := writeLines(filepath.Join(cfg.OutDir, "filtered_not.txt"), filteredNot); err != nil {
		fmt.Println(Red+"Erreur écriture filtered_not.txt :"+Reset, err)
		return
	}

	headLines := head(lines, n)
	if err := writeLines(filepath.Join(cfg.OutDir, "head.txt"), headLines); err != nil {
		fmt.Println(Red+"Erreur écriture head.txt :"+Reset, err)
		return
	}

	tailLines := tail(lines, n)
	if err := writeLines(filepath.Join(cfg.OutDir, "tail.txt"), tailLines); err != nil {
		fmt.Println(Red+"Erreur écriture tail.txt :"+Reset, err)
		return
	}
	fmt.Println("Choix terminé. Résultats écrits dans", cfg.OutDir, ": filtered.txt, filtered_not.txt, head.txt, tail.txt")
}

func runChoiceB(reader *bufio.Reader, cfg Config) {
	dir := readLine(reader, "Répertoire à analyser (Entrée = défaut) : ")
	if dir == "" {
		dir = cfg.BaseDir
	}

	info, err := os.Stat(dir)
	if err != nil {
		fmt.Println("Erreur : répertoire introuvable :", dir)
		return
	}
	if !info.IsDir() {
		fmt.Println("Erreur : ce chemin n'est pas un répertoire :", dir)
		return
	}

	files, err := listFilesWithExt(dir, cfg.DefaultExt)
	if err != nil {
		fmt.Println("Erreur lecture répertoire :", err)
		return
	}
	if len(files) == 0 {
		fmt.Println("Aucun fichier", cfg.DefaultExt, "trouvé dans", dir)
		return
	}

	var summaries []FileSummary
	for _, p := range files {
		summaries = append(summaries, analyzeOneFile(p))
	}

	reportPath := filepath.Join(cfg.OutDir, "report.txt")
	if err := writeReport(reportPath, dir, cfg.DefaultExt, summaries); err != nil {
		fmt.Println("Erreur écriture report.txt :", err)
		return
	}

	indexPath := filepath.Join(cfg.OutDir, "index.txt")
	if err := writeIndex(indexPath, summaries); err != nil {
		fmt.Println("Erreur écriture index.txt :", err)
		return
	}

	mergedPath := filepath.Join(cfg.OutDir, "merged.txt")
	if err := mergeFiles(mergedPath, summaries); err != nil {
		fmt.Println("Erreur écriture merged.txt :", err)
		return
	}

	fmt.Println("Choix B terminé. Fichiers générés dans", cfg.OutDir, ": report.txt, index.txt, merged.txt")
}

func runChoiceC(reader *bufio.Reader, cfg Config) {
	article := readLine(reader, "Nom de l'article Wikipédia (ex: Go_(langage)) : ")
	if article == "" {
		fmt.Println(Red + "Article invalide." + Reset)
		return
	}

	url := "https://fr.wikipedia.org/wiki/" + article
	fmt.Println("Téléchargement :", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Erreur création requête :", err)
		return
	}

	req.Header.Set("User-Agent", "GoWikiBot/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Erreur téléchargement :", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println("Erreur HTTP :", resp.Status)
		return
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println("Erreur parsing HTML :", err)
		return
	}

	var paragraphs []string

	doc.Find("#mw-content-text p").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			paragraphs = append(paragraphs, text)
		}
	})

	if len(paragraphs) == 0 {
		fmt.Println(Red + "Aucun contenu trouvé." + Reset)
		return
	}

	fmt.Println("Nombre de paragraphes récupérés :", Green, len(paragraphs), Reset)

	totalWords, avgLen := wordStats(paragraphs)

	fmt.Println("Stats mots Wikipédia :")
	fmt.Println("Nombre de mots :", Green, totalWords, Reset)
	fmt.Printf("Longueur moyenne : %s%.2f%s\n", Green, avgLen, Reset)

	keyword := readLine(reader, "Mot-clé à filtrer : ")
	filtered := filterLines(paragraphs, keyword, true)

	outputPath := filepath.Join(cfg.OutDir, "wiki_"+article+".txt")

	err = writeLines(outputPath, filtered)
	if err != nil {
		fmt.Println(Red+"Erreur écriture fichier :", err, Reset)
		return
	}

	fmt.Println("Résultats écrits dans :", Green, outputPath, Reset)
}

func listFilesWithExt(dir string, ext string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var out []string
	extLower := strings.ToLower(ext)

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		nameLower := strings.ToLower(e.Name())
		if strings.HasSuffix(nameLower, extLower) {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out, nil
}

func analyzeOneFile(path string) FileSummary {
	s := FileSummary{Path: path}

	info, err := os.Stat(path)
	if err != nil {
		s.Err = err
		return s
	}

	s.SizeBytes = info.Size()
	s.ModTime = info.ModTime().String()

	lines, err := readAllLines(path)
	if err != nil {
		s.Err = err
		return s
	}
	s.Lines = len(lines)

	words, avg := wordStats(lines)
	s.Words = words
	s.AvgWordLn = avg

	return s
}

func writeReport(path string, dir string, ext string, summaries []FileSummary) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	var totalSize int64
	var totalLines, totalWords int
	totalFiles := len(summaries)
	okFiles := 0

	_, _ = w.WriteString("=== REPORT GLOBAL ===\n")
	_, _ = w.WriteString("Directory: " + dir + "\n")
	_, _ = w.WriteString("Extension: " + ext + "\n\n")

	for _, s := range summaries {
		_, _ = w.WriteString("File: " + s.Path + "\n")

		if s.Err != nil {
			_, _ = w.WriteString("  ERROR: " + s.Err.Error() + "\n\n")
			continue
		}

		okFiles++
		totalSize += s.SizeBytes
		totalLines += s.Lines
		totalWords += s.Words

		_, _ = w.WriteString(fmt.Sprintf("  Size(bytes): %d\n", s.SizeBytes))
		_, _ = w.WriteString("  ModTime: " + s.ModTime + "\n")
		_, _ = w.WriteString(fmt.Sprintf("  Lines: %d\n", s.Lines))
		_, _ = w.WriteString(fmt.Sprintf("  Words(no-numeric): %d\n", s.Words))
		_, _ = w.WriteString(fmt.Sprintf("  AvgWordLen: %.2f\n", s.AvgWordLn))
		_, _ = w.WriteString("\n")
	}

	_, _ = w.WriteString("=== TOTALS ===\n")
	_, _ = w.WriteString(fmt.Sprintf("Files found: %d\n", totalFiles))
	_, _ = w.WriteString(fmt.Sprintf("Files analyzed OK: %d\n", okFiles))
	_, _ = w.WriteString(fmt.Sprintf("Total size(bytes): %d\n", totalSize))
	_, _ = w.WriteString(fmt.Sprintf("Total lines: %d\n", totalLines))
	_, _ = w.WriteString(fmt.Sprintf("Total words(no-numeric): %d\n", totalWords))

	return nil
}

func writeIndex(path string, summaries []FileSummary) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	_, _ = w.WriteString("path\tsize_bytes\tmod_time\n")

	for _, s := range summaries {
		if s.Err != nil {
			_, _ = w.WriteString(fmt.Sprintf("%s\tERROR\t%s\n", s.Path, s.Err.Error()))
			continue
		}
		_, _ = w.WriteString(fmt.Sprintf("%s\t%d\t%s\n", s.Path, s.SizeBytes, s.ModTime))
	}

	return nil
}

func mergeFiles(outPath string, summaries []FileSummary) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	for _, s := range summaries {
		if s.Err != nil {
			continue
		}

		_, _ = w.WriteString("===== " + s.Path + " =====\n")

		lines, err := readAllLines(s.Path)
		if err != nil {
			_, _ = w.WriteString("ERROR reading file: " + err.Error() + "\n\n")
			continue
		}

		for _, line := range lines {
			_, _ = w.WriteString(line + "\n")
		}
		_, _ = w.WriteString("\n")
	}

	return nil
}

func printFileInfos(path string, nbLines int) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	fmt.Println("\nInfos fichier :")
	fmt.Println("Chemin :", Green, path, Reset)
	fmt.Println("Taille en octets :", Green, info.Size(), Reset)
	fmt.Println("Date modification :", Green, info.ModTime().Format("02-01-2006 15:04:05"), Reset)
	fmt.Println("Nombre de lignes :", Green, nbLines, Reset)

	return nil
}

func readAllLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func wordStats(lines []string) (int, float64) {
	totalWords := 0
	totalLen := 0

	for _, line := range lines {
		words := strings.Fields(line)
		for _, w := range words {
			if isNumericWord(w) {
				continue
			}
			totalWords++
			totalLen += len([]rune(w))
		}
	}

	if totalWords == 0 {
		return 0, 0
	}
	return totalWords, float64(totalLen) / float64(totalWords)
}

func isNumericWord(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func countLinesWithKeyword(lines []string, keyword string) int {
	if keyword == "" {
		return 0
	}
	count := 0
	for _, line := range lines {
		if strings.Contains(line, keyword) {
			count++
		}
	}
	return count
}

func filterLines(lines []string, keyword string, keepIfContains bool) []string {
	var out []string
	for _, line := range lines {
		contains := keyword != "" && strings.Contains(line, keyword)
		if keepIfContains && contains {
			out = append(out, line)
		}
		if !keepIfContains && !contains {
			out = append(out, line)
		}
	}
	return out
}

func head(lines []string, n int) []string {
	if n <= 0 {
		return []string{}
	}
	if n > len(lines) {
		n = len(lines)
	}
	return lines[:n]
}

func tail(lines []string, n int) []string {
	if n <= 0 {
		return []string{}
	}
	if n > len(lines) {
		n = len(lines)
	}
	return lines[len(lines)-n:]
}

func writeLines(path string, lines []string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, line := range lines {
		if _, err := w.WriteString(line + "\n"); err != nil {
			return err
		}
	}
	return w.Flush()
}

func readLine(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func readInt(reader *bufio.Reader, prompt string) int {
	for {
		s := readLine(reader, prompt)
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err == nil {
			return n
		}
		fmt.Println("Veuillez entrer un nombre valide.")
	}
}
