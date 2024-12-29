package internal

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hhatto/gocloc"
)

func skipProjects(projects []fs.DirEntry, out string) []fs.DirEntry {
	existingResults := make(map[string]struct{})

	addToMap := func(dir string) {
		files, err := os.ReadDir(dir)
		if err != nil {
			log.Fatalf("Error reading directory %s: %v", dir, err)
		}
		for _, file := range files {
			if !file.IsDir() {
				existingResults[strings.TrimSuffix(file.Name(), ".json")] = struct{}{}
			}
		}
	}

	addToMap(out)
	addToMap(filepath.Join(out, "no_findings"))

	var filteredProjects []fs.DirEntry
	for _, project := range projects {
		if project.IsDir() {
			if _, exists := existingResults[project.Name()]; !exists {
				filteredProjects = append(filteredProjects, project)
			}
		}
	}
	fmt.Printf("Skipping %d projects with existing results\n", len(projects)-len(filteredProjects))
	return filteredProjects
}

func reportProgress(completed *int64, total int64) {
	for {
		completedItems := atomic.LoadInt64(completed)
		percentComplete := float64(completedItems) / float64(total) * 100
		fmt.Printf("\rProgress: %.2f%% (%d/%d)", percentComplete, completedItems, total)

		if completedItems == total {
			fmt.Println()
			return
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func getJSONFiles(dir string) []os.DirEntry {
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("Error reading directory %s: %v", dir, err)
	}
	return files
}

func countJSONFiles(files []os.DirEntry) int {
	count := 0
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			count++
		}
	}
	return count
}

func getLinesOfCode(dir, language string) int {
	languages := gocloc.NewDefinedLanguages()
	options := gocloc.NewClocOptions()
	paths := []string{
		dir,
	}

	processor := gocloc.NewProcessor(languages, options)
	result, err := processor.Analyze(paths)
	if err != nil {
		log.Fatalf("gocloc fail. error: %v", err)
		return -1
	}

	for _, lang := range result.Languages {
		if strings.EqualFold(lang.Name, language) {
			return int(lang.Code)
		}
	}
	log.Fatalf("Zero lines of code found in the selected language, exiting...")
	os.Exit(1)
	return 0
}

func ReadTestInfoFromFile(filePath string) ([]TestInfo, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var testInfos []TestInfo
	err = json.Unmarshal(file, &testInfos)
	if err != nil {
		return nil, err
	}

	return testInfos, nil
}
