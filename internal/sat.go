package internal

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

func EvalProjects(testInfoPath, resultsDir string) error {

	testInfo, err := ReadTestInfoFromFile(testInfoPath)
	if err != nil {
		log.Fatalf("Error reading test info: %v", err)
	}

	err = CombineResults(resultsDir, testInfo)
	if err != nil {
		log.Fatalf("Error combining results: %v", err)
	}
	return nil
}

func EvalRule(rulesPath, targetsPath, out, language string, print, multi bool) error {

	fileInfo, err := os.Stat(targetsPath)
	if err != nil {
		log.Fatalf("Error accessing directory: %v", err)
	}
	if !fileInfo.IsDir() {
		log.Fatalf("Provided path is not a directory")
	}

	outNoFindings := out + "/no_findings"
	if _, err := os.Stat(outNoFindings); os.IsNotExist(err) {
		os.Mkdir(outNoFindings, 0755)
	}

	projects, err := os.ReadDir(targetsPath)
	if err != nil {
		log.Fatalf("Error reading directory: %v", err)
	}
	projects = skipProjects(projects, out)

	fmt.Printf("Analyzing %d projects...\n", len(projects))

	if multi {
		numWorkers := runtime.NumCPU()
		fmt.Println("Number of Workers:", numWorkers)

		projectsChan := make(chan fs.DirEntry, len(projects))
		var wg sync.WaitGroup

		var completed int64
		totalItems := int64(len(projects))

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go worker(&wg, projectsChan, rulesPath, targetsPath, out, print, &completed)
		}

		go reportProgress(&completed, totalItems)

		for _, target := range projects {
			projectsChan <- target
		}
		close(projectsChan)
		wg.Wait()
	} else {
		current := 1
		for _, target := range projects {
			if target.IsDir() {
				fmt.Printf("Analyzing %s [%d\\%d]\n", target.Name(), current, len(projects))

				err := analyze(rulesPath, target.Name(), targetsPath, out, print)
				if err != nil {
					log.Fatalf("Error analyzing project %s: %v", target.Name(), err)
				}
				current++
			}
		}
	}
	analyzeResults(targetsPath, out, language)
	return nil
}

func processFile(file os.DirEntry, projects, out, language string, wg *sync.WaitGroup, semaphore chan struct{}, resultsChan chan<- SingleProjectResults, processedFiles *atomic.Int32, totalFiles int) {
	defer wg.Done()
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	fp := filepath.Join(out, file.Name())
	byteValue, err := os.ReadFile(fp)
	if err != nil {
		log.Printf("Error reading file %s: %v", fp, err)
		return
	}

	var resultsFile SemgrepResultsFile
	err = json.Unmarshal(byteValue, &resultsFile)
	if err != nil {
		log.Printf("Error unmarshalling file %s: %v", fp, err)
		return
	}

	projectName := file.Name()[:strings.Index(file.Name(), ".json")]
	findings := len(resultsFile.Results)
	linesOfCode := getLinesOfCode(filepath.Join(projects, projectName), language)
	resultsChan <- SingleProjectResults{
		ProjectName: projectName,
		Findings:    findings,
		LinesOfCode: linesOfCode,
		Ratio:       float64(findings) / float64(linesOfCode),
	}

	processed := processedFiles.Add(1)
	fmt.Printf("\rProgress: %d/%d files processed (%.2f%%)", processed, totalFiles, float64(processed)/float64(totalFiles)*100)
}
