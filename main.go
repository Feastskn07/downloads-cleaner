package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Categories map[string]string

var defaultCategories = Categories{
	".jpg":  "images",
	".jpeg": "images",
	".png":  "images",
	".gif":  "images",

	".mp4": "videos",
	".mkv": "videos",
	".avi": "videos",
	".mov": "videos",

	".mp3": "audio",
	".wav": "audio",

	".pdf":  "documents",
	".txt":  "documents",
	".doc":  "documents",
	".docx": "documents",
	".xlsx": "documents",

	".zip": "archives",
	".rar": "archives",
	".7z":  "archives",

	".exe": "applications",
	".msi": "applications",
}

func main() {
	// How to use
	// go run main.go -dir "C:\Users\Feas\Downloads" -dryrun -include-dirs -config ".\categories.json" -log ".\organizer.log"

	// Define command-line flags
	dirFlag := flag.String("dir", "", "Düzenlenecek klasör yolu (Zorunlu!)")
	dryRunFlag := flag.Bool("dryrun", false, "True ise dosyalar taşınmaz, sadece ne yapılacağını gösterir.")
	configPath := flag.String("config", "", "Opsiyonel. Özelleştirilmiş kategori yapılandırma dosyasının yolu.")
	includeSubdirs := flag.Bool("subdirs", false, "True ise alt klasörler de taranır.")
	logPath := flag.String("log", "organizer.log", "Log dosyası yolu (stdout ile beraber yazılır.).")

	flag.Parse()

	// Check if the required 'dir' flag is provided
	if *dirFlag == "" {
		fmt.Println("Hata: -dir parametresi zorunludur. Örnek kullanım:")
		fmt.Println("  go run main.go -dir \"C:\\Users\\User\\Downloads\" -dryrun")
		os.Exit(1)
	}

	logger, closer, err := openLogger(*logPath)
	if err != nil {
		fmt.Println("Hata: Log dosyası açılamadı: ", err)
		os.Exit(1)
	}
	defer closer.Close()

	logger.Println("[INFO] Başladı")
	logger.Printf("[INFO] Klasör: %s, Dry Run: %v, Alt Klasörler: %v, Konfigürasyon: %s\n", *dirFlag, *dryRunFlag, *includeSubdirs, *configPath)

	cats, catErr := readCategories(*configPath)
	if catErr != nil {
		logger.Printf("[WARN] Kategori dosyası okunamadı, varsayılan kategoriler kullanılacak: %v\n", catErr)
	}

	// Check if the provided directory exists
	info, err := os.Stat(*dirFlag)
	if err != nil {
		fmt.Println("Hata: Verilen klasöre erişilemiyor: ", err)
		os.Exit(1)
	}

	if !info.IsDir() {
		fmt.Println("Hata: Verilen yol bir klasör değil.")
		os.Exit(1)
	}

	// Rightnow we just report the received parameters
	fmt.Println("Düzenlenecek klasör yolu:", *dirFlag)
	fmt.Println("Dry Run Modu:", *dryRunFlag)

	// List files in the directory
	entries, err := listFiles(*dirFlag)
	if err != nil {
		fmt.Println("Hata: Klasördeki dosyalar listelenemedi: ", err)
		os.Exit(1)
	}

	fmt.Println("Planlama: ")
	for _, e := range entries {
		if e.IsDir() {
			fmt.Println("  [Klasör] ", e.Name())
			continue
		}

		fileName := e.Name()
		category := getTargetFolder(fileName, cats)

		destDir := filepath.Join(*dirFlag, category)
		previewPath := filepath.Join(destDir, fileName)

		if *dryRunFlag {
			// Just preview
			fmt.Printf("  [Dry Run] %s -> %s\n", fileName, previewPath)
			continue
		}

		// Move the file
		finalPath, err := moveFileToCategory(*dirFlag, fileName, cats)
		if err != nil {
			fmt.Printf("  [Hata] %s taşınamadı: %v\n", fileName, err)
			continue
		}

		fmt.Printf("  [Taşındı] %s -> %s\n", fileName, finalPath)
	}
}

func openLogger(logFilePath string) (*log.Logger, io.Closer, error) {
	f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}
	mw := io.MultiWriter(os.Stdout, f)
	logger := log.New(mw, "", log.LstdFlags)
	return logger, f, nil
}

func readCategories(path string) (Categories, error) {
	if path == "" {
		return defaultCategories, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		// If error reading file, return default categories
		return defaultCategories, err
	}
	var c Categories
	if err := json.Unmarshal(b, &c); err != nil {
		// If error parsing JSON, return default categories
		return defaultCategories, err
	}
	return c, nil
}

func listFiles(dir string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func getTargetFolder(fileName string, cats Categories) string {
	ext := strings.ToLower(filepath.Ext(fileName)) // ".PNG" -> ".png"
	if folder, ok := cats[ext]; ok {
		return folder
	}
	return "others"
}

func uniquePath(dir, baseName string) (string, error) {
	ext := filepath.Ext(baseName)                 // ".png"
	nameOnly := strings.TrimSuffix(baseName, ext) // "photo"

	candidate := filepath.Join(dir, baseName)
	counter := 2

	for {
		// os.Stat returns error if the file does not exist
		_, err := os.Stat(candidate)
		if os.IsNotExist(err) {
			return candidate, nil // File does not exist, return this path
		}

		// File exists, generate a new candidate
		newName := fmt.Sprintf("%s(%d)%s", nameOnly, counter, ext)
		candidate = filepath.Join(dir, newName)
		counter++
	}
}

func moveFileToCategory(downloadsDir, fileName string, cats Categories) (string, error) {
	srcPath := filepath.Join(downloadsDir, fileName)

	// Determine target folder
	categoryFolder := getTargetFolder(fileName, cats)

	// Target directory path
	destDir := filepath.Join(downloadsDir, categoryFolder)

	// If the target directory does not exist, create it
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}

	// Determine unique destination path
	finalDestPath, err := uniquePath(destDir, fileName)
	if err != nil {
		return "", err
	}

	// Move the file
	if err := os.Rename(srcPath, finalDestPath); err != nil {
		return "", err
	}

	return finalDestPath, nil
}
