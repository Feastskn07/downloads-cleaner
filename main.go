package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var categories = map[string]string{
	".jpg":   "images",
	".jpeg":  "images",
	".png":   "images",
	".gif":   "images",

	".mp4":  "videos",
	".mkv":  "videos",
	".avi":  "videos",
	".mov":  "videos",

	".mp3":  "audio",
	".wav":  "audio",

	".pdf":  "documents",
	".txt":  "documents",
	".doc":  "documents",
	".docx": "documents",
	".xlsx": "documents",

	".zip":  "archives",
	".rar":  "archives",
	".7z":   "archives",

	".exe":  "applications",
	".msi":  "applications",
}

func main() {
	// Define command-line flags
	dirFlag := flag.String("dir", "", "Düzenlenecek klasör yolu (Zorunlu!)")
	dryRunFlag := flag.Bool("dryrun", false, "True ise dosyalar taşınmaz, sadece ne yapılacağını gösterir.")

	flag.Parse()

	// Check if the required 'dir' flag is provided
	if *dirFlag == "" {
		fmt.Println("Hata: -dir parametresi zorunludur. Örnek kullanım:")
		fmt.Println("  go run main.go -dir \"C:\\Users\\User\\Downloads\" -dryrun")
		os.Exit(1)
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
		category := getTargetFolder(fileName)

		destDir := filepath.Join(*dirFlag, category)
		destPath := filepath.Join(destDir, fileName)

		if *dryRunFlag {
			fmt.Printf("  [Dry Run] Taşınacak: %s -> %s\n", fileName, destPath)
		} else {
			// Here we would move the file (not implemented yet)
			fmt.Printf("  Taşınıyor: %s -> %s\n", fileName, destPath)
		}
	}
}

// listFiles: Scans the given directory and returns a list of files.
// For now, just scan the main directory and not subdirectories.
func listFiles(dir string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// getTargetFolder: Returns the target folder name based on file extension.
func getTargetFolder(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName)) // ".PNG" -> ".png"
	if folder, ok := categories[ext]; ok {
		return folder
	}
	return "others"
}