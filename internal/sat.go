package internal

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func EvalProject(targetPath, rulesPath string) error {
	return nil
}

func EvalRule(rulesPath, targetsPath, out string, print, multi bool) error {

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
		numWorkers := runtime.NumCPU() * 2
		fmt.Println("Number of CPUs:", numWorkers / 2)

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
	return nil
}

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
	fmt.Printf("Skipping %d projects with existing results\n", len(projects) - len(filteredProjects))
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

func worker(wg *sync.WaitGroup, projects <- chan fs.DirEntry, rulesPath, targetsPath, out string, print bool, completed *int64) {
	defer wg.Done()
	for target := range projects {
		if target.IsDir() {
			err := analyze(rulesPath, target.Name(), targetsPath, out, print)
			if err != nil {
				log.Fatalf("Error analyzing project %s: %v", target.Name(), err)
			}
		}
		atomic.AddInt64(completed, 1)
	}
}

func analyze(rulesPath, target, rootDir, out string, print bool) error {
	fullTargetpath := fmt.Sprintf("%s/%s", rootDir, target)
	outputFile := fmt.Sprintf("%s/%s.json", out, target)

	semgrep := exec.Command("semgrep", "--json-output", outputFile, "--config", rulesPath, "--", fullTargetpath)

	var outputBuffer bytes.Buffer
	
	if print {
		multiWrite := io.MultiWriter(os.Stdout, &outputBuffer)
		semgrep.Stdout = multiWrite
		semgrep.Stderr = multiWrite
	} else {
		semgrep.Stdout = &outputBuffer
		semgrep.Stderr = &outputBuffer
	}

	err := semgrep.Run()
	if err != nil {
		return fmt.Errorf("semgrep execution failed: %w", err)
	}

	strOut := outputBuffer.String()
	findingsIndex := strings.Index(strOut, "finding")
	if findingsIndex == -1 {
		return fmt.Errorf("failed to find 'findings' in output")
	} else {
		findings := strOut[findingsIndex -2:findingsIndex -1]
		numFindings, err := strconv.Atoi(findings)
		if err != nil {
			return fmt.Errorf("failed to convert findings to integer: %w", err)
		}
		if numFindings == 0 {
			newPath := out + "/no_findings/" + target + ".json"
			_ = os.Rename(outputFile, newPath)
		}
		return nil
	}
}