package main

import (
	"flag"
	"fmt"
	"os"
)

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

	fmt.Println("Bulunan ögeler: ")
	for _, e := range entries {
		// Is it a file or directory?
		if e.IsDir() {
			fmt.Println("  [Klasör] ", e.Name())
		} else {
			fmt.Println("  [Dosya ] ", e.Name())
		}
	}

	// TODO: Next steps:
	// 1. Scan the directory for files
	// 2. Categorize files based on their extensions
	// 3. If dry run is true, print the planned moves
	// 4. If dry run is false, move the files to their respective folders
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