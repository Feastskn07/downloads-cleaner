package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	gui := flag.Bool("gui", false, "Masaüstü arayüzünü başlatır.")

	flag.Parse()

	// Check if the required 'dir' flag is provided
	if *dirFlag == "" {
		fmt.Println("Hata: -dir parametresi zorunludur. Örnek kullanım:")
		fmt.Println("  go run main.go -dir \"C:\\Users\\User\\Downloads\" -dryrun")
		os.Exit(1)
	}

	if *gui {
		startGUI()
		return
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

	managed := makeManagedSet(cats)

	fileList, err := collectFiles(*dirFlag, *includeSubdirs, managed)
	if err != nil {
		logger.Printf("[ERROR] Dosyalar toplanamadı: %v\n", err)
		os.Exit(1)
	}

	if len(fileList) == 0 {
		logger.Println("[INFO] Düzenlenecek dosya bulunamadı.")
		return
	}

	logger.Printf("[INFO] Toplam %d dosya bulundu.\n", len(fileList))

	for _, rel := range fileList {
		base := filepath.Base(rel)
		category := getTargetFolder(base, cats)
		previewDest := filepath.Join(*dirFlag, category, base)

		if *dryRunFlag {
			logger.Printf("[DRY RUN] %s -> %s\n", rel, previewDest)
			continue
		}

		finalPath, err := moveFileToCategoryFromPath(*dirFlag, rel, cats)
		if err != nil {
			logger.Printf("[ERROR] %s taşınamadı: %v\n", rel, err)
			continue
		}
		logger.Printf("[MOVED] %s -> %s\n", rel, finalPath)
	}

	logger.Println("[INFO] İşlem tamamlandı.")

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

func startGUI() {
	a := app.New()
	w := a.NewWindow("Downloads Cleaner")
	w.Resize(fyne.NewSize(400, 300))

	dirEntry := widget.NewEntry()
	dirEntry.SetPlaceHolder(`Örn: C:\Users\user\Downloads`)
	btnPick := widget.NewButton("Klasör Seç", func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			p := uri.Path()
			dirEntry.SetText(p)
		}, w)
		d.SetLocation(storage.NewFileURI("file:///"))
		d.Show()
	})

	chkSub := widget.NewCheck("Alt klasörleri tarasın", func(bool) {})
	chkDry := widget.NewCheck("Dry Run Modu", func(bool) {})
	configEntry := widget.NewEntry()
	configEntry.SetPlaceHolder(`Opsiyonel: C:\path\to\categories.json`)
	btnPickCfg := widget.NewButton("Config Seç", func() {
		d := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
			if err != nil || rc == nil {
				return
			}
			configEntry.SetText(rc.URI().Path())
			rc.Close()
		}, w)
		d.Show()
	})

	list := widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject { return widget.NewLabel("...") },
		func(i widget.ListItemID, o fyne.CanvasObject) {},
	)
	var previewRows []string
	list.Length = func() int { return len(previewRows) }
	list.UpdateItem = func(i widget.ListItemID, o fyne.CanvasObject) {
		o.(*widget.Label).SetText(previewRows[i])
	}

	logArea := widget.NewMultiLineEntry()
	logArea.SetPlaceHolder("Loglar burada görünecek...")
	logArea.Wrapping = fyne.TextWrapWord
	logArea.SetMinRowsVisible(8)

	status := widget.NewLabel("")
	btnPreview := widget.NewButton("Önizleme", nil)
	btnRun := widget.NewButton("Taşı", nil)

	topRow := container.NewBorder(nil, nil,
		container.NewHBox(widget.NewLabel("Klasör:"), dirEntry, btnPick),
		nil, nil)

	cfgRow := container.NewHBox(widget.NewLabel("Config:"), configEntry, btnPickCfg)
	optsRow := container.NewHBox(chkSub, chkDry)

	left := container.NewVBox(topRow, cfgRow, optsRow, container.NewHBox(btnPreview, btnRun), status)
	split := container.NewHSplit(left, container.NewVSplit(list, logArea))
	split.Offset = 0.4

	w.SetContent(split)

	appendLog := func(line string) {
		ts := time.Now().Format("15:04:05")
		logArea.SetText(logArea.Text + "[" + ts + "]" + line + "\n")
		logArea.CursorRow = len(logArea.Text)
	}

	btnPreview.OnTapped = func() {
		rootIn := dirEntry.Text
		resolved, err := resolveDownloadsDir(rootIn)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		cats, cerr := readCategories(configEntry.Text)
		if cerr != nil {
			appendLog("[WARN] Kategori dosyası okunamadı, varsayılan kategoriler kullanılacak." + cerr.Error())
		}
		managed := makeManagedSet(cats)

		status.SetText("Önizleme hazırlanıyor...")
		previewRows = nil
		list.Refresh()

		go func() {
			files, ferr := collectFiles(resolved, chkSub.Checked, managed)
			if ferr != nil {
				fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "Hata", Content: ferr.Error()})
				fyne.CurrentApp().Driver().CallOnMain(func() {
					status.SetText("Hata: " + ferr.Error())
				})
				return
			}
			var out []string
			for _, rel := range files {
				base := filepath.Base(rel)
				cat := getTargetFolder(base, cats)
				out = append(out, rel+" -> "+filepath.Join(resolved, cat, base))
			}
			fyne.CurrentApp().Driver().CallOnMain(func() {
				previewRows = out
				list.Refresh()
				status.SetText(fmt.Sprintf("Toplam %d dosya bulunuyor.", len(out)))
				appendLog(fmt.Sprintf("[INFO] Toplam %d dosya bulundu.", len(out)))
			})
		}()
	}

	btnRun.OnTapped = func() {
		rootIn := dirEntry.Text
		resolved, err := resolveDownloadsDir(rootIn)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		cats, cerr := readCategories(configEntry.Text)
		if cerr != nil {
			appendLog("[WARN] Kategori dosyası okunamadı, varsayılan kategoriler kullanılacak." + cerr.Error())
		}
		managed := makeManagedSet(cats)

		status.SetText("Dosyalar taşınıyor...")
		btnRun.Disable()
		btnPreview.Disable()

		go func() {
			files, ferr := collectFiles(resolved, chkSub.Checked, managed)
			if ferr != nil {
				fyne.CurrentApp().Driver().CallOnMain(func() {
					dialog.ShowError(ferr, w)
					status.SetText("Hata: " + ferr.Error())
					btnRun.Enable()
					btnPreview.Enable()
				})
				return
			}

			moved, errs := 0, 0
			dry := chkDry.Checked

			for _, rel := range files {
				base := filepath.Base(rel)
				cat := getTargetFolder(base, cats)
				dest := filepath.Join(resolved, cat, base)

				if dry {
					fyne.CurrentApp().Driver().CallOnMain(func() {
						appendLog("[DRY RUN] " + rel + " -> " + dest)
					})
					continue
				}

				finalPath, merr := moveFileToCategoryFromPath(resolved, rel, cats)
				if merr != nil {
					errs++
					fyne.CurrentApp().Driver().CallOnMain(func() {
						appendLog("[ERROR] " + rel + " taşınamadı: " + merr.Error())
					})
					continue
				}
				moved++
				fyne.CurrentApp().Driver().CallOnMain(func() {
					appendLog("[MOVED] " + rel + " -> " + finalPath)
				})
			}

			fyne.CurrentApp().Driver().CallOnMain(func() {
				status.SetText(fmt.Sprintf("İşlem tamamlandı. %d taşındı, %d hata.", moved, errs))
				btnRun.Enable()
				btnPreview.Enable()
				previewRows = nil
				list.Refresh()
			})
		}()
	}
	w.ShowAndRun()
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

func makeManagedSet(cats Categories) map[string]bool {
	set := map[string]bool{"others": true}
	// TODO: Use maps package
	for _, v := range cats {
		set[v] = true
	}
	return set
}

func collectFiles(root string, includeSubdirs bool, manageDirs map[string]bool) ([]string, error) {
	var files []string

	if !includeSubdirs {
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() {
				files = append(files, e.Name())
			}
		}
		return files, nil
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}

		rel, _ := filepath.Rel(root, path)

		if d.IsDir() {
			base := filepath.Base(path)
			if manageDirs[base] {
				return filepath.SkipDir
			}
			return nil
		}

		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
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

func moveFileToCategoryFromPath(root, rel string, cats Categories) (string, error) {
	srcPath := filepath.Join(root, rel)
	fileName := filepath.Base(rel)

	categoryFolder := getTargetFolder(fileName, cats)
	destDir := filepath.Join(root, categoryFolder)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}
	finalDestPath, err := uniquePath(destDir, fileName)
	if err != nil {
		return "", err
	}
	if err := os.Rename(srcPath, finalDestPath); err != nil {
		return "", err
	}
	return finalDestPath, nil
}
