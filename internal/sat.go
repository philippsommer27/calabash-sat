package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hhatto/gocloc"
)

func EvalProject(targetPath, rulesPath, out string) error {
	return analyze(rulesPath, targetPath, "", out, true)
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
	numFindings, err := getResultsCount(outputFile)
	if err != nil {
		return fmt.Errorf("error reading results file: %w", err)
	}
	if numFindings == 0 {
		newPath := out + "/no_findings/" + target + ".json"
		_ = os.Rename(outputFile, newPath)
	}
	return nil
}

func getResultsCount(filename string) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		return 0, err
	}

	var resultsFile SemgrepResultsFile
	err = json.Unmarshal(byteValue, &resultsFile)
	if err != nil {
		return 0, err
	}

	return len(resultsFile.Results), nil
}

type EvalRulesResults struct {
    TotalFindings   int
    TotalProjects   int
    ProjectFindings []SingleProjectResults
}

type SingleProjectResults struct {
    ProjectName string
    Findings    int
    LinesOfCode int
    Ratio       float64
    Percentile  float64
    Grade       string
}

func analyzeResults(projects, out, language string) {
    files := getJSONFiles(out)
    results := processFiles(files, projects, out, language)
	thresholds := calculatePercentagesAndGrades(&results)
    sortResults(&results)
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		log.Fatalf("Error marshalling results to JSON: %v", err)
	}

	resultsFile := filepath.Join(out, "_sat_results.json")
	err = os.WriteFile(resultsFile, resultsJSON, 0644)
	if err != nil {
		log.Fatalf("Error writing results to file: %v", err)
	}

	err = writeThresholdsToFile(thresholds, out)
    if err != nil {
        log.Printf("Warning: Failed to write grade thresholds: %v", err)
    }

	fmt.Printf("Results written\n",)
}

func getJSONFiles(dir string) []os.DirEntry {
    files, err := os.ReadDir(dir)
    if err != nil {
        log.Fatalf("Error reading directory %s: %v", dir, err)
    }
    return files
}

func processFiles(files []os.DirEntry, projects, out, language string) EvalRulesResults {
    var results EvalRulesResults
    var wg sync.WaitGroup
    resultsChan := make(chan SingleProjectResults, len(files))
    semaphore := make(chan struct{}, runtime.NumCPU())

    totalFiles := countJSONFiles(files)
    processedFiles := atomic.Int32{}

    for _, file := range files {
        if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
            wg.Add(1)
            go processFile(file, projects, out, language, &wg, semaphore, resultsChan, &processedFiles, totalFiles)
        }
    }

    go func() {
        wg.Wait()
        close(resultsChan)
    }()

    for result := range resultsChan {
        results.ProjectFindings = append(results.ProjectFindings, result)
        results.TotalFindings += result.Findings
        results.TotalProjects++
    }

    fmt.Printf("\nTotal projects analyzed: %d\n", results.TotalProjects)
    return results
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

func calculatePercentagesAndGrades(results *EvalRulesResults) GradeThresholds {
    sort.Slice(results.ProjectFindings, func(i, j int) bool {
        return results.ProjectFindings[i].Ratio < results.ProjectFindings[j].Ratio
    })

    totalProjects := len(results.ProjectFindings)
    thresholds := GradeThresholds{
        A_plus: results.ProjectFindings[int(float64(totalProjects)*0.1)].Ratio,
        A: results.ProjectFindings[int(float64(totalProjects)*0.2)].Ratio,
        B: results.ProjectFindings[int(float64(totalProjects)*0.4)].Ratio,
        C: results.ProjectFindings[int(float64(totalProjects)*0.6)].Ratio,
        D: results.ProjectFindings[int(float64(totalProjects)*0.8)].Ratio,
        F: results.ProjectFindings[int(float64(totalProjects)*0.9)].Ratio,
    }

    for i := range results.ProjectFindings {
        results.ProjectFindings[i].Percentile = float64(i) / float64(totalProjects-1) * 100
        results.ProjectFindings[i].Grade = getGrade(results.ProjectFindings[i].Percentile)
    }

    return thresholds
}

func writeThresholdsToFile(thresholds GradeThresholds, outputDir string) error {
    thresholdsJSON, err := json.MarshalIndent(thresholds, "", "  ")
    if err != nil {
        return fmt.Errorf("error marshalling thresholds to JSON: %v", err)
    }

    thresholdsFile := filepath.Join(outputDir, "_grade_thresholds.json")
    err = os.WriteFile(thresholdsFile, thresholdsJSON, 0644)
    if err != nil {
        return fmt.Errorf("error writing thresholds to file: %v", err)
    }

    fmt.Printf("Grade thresholds written to %s\n", thresholdsFile)
    return nil
}

func sortResults(results *EvalRulesResults) {
    sort.Slice(results.ProjectFindings, func(i, j int) bool {
        if results.ProjectFindings[i].Grade != results.ProjectFindings[j].Grade {
            return results.ProjectFindings[i].Grade < results.ProjectFindings[j].Grade
        }
        return results.ProjectFindings[i].Ratio < results.ProjectFindings[j].Ratio
    })
}

func getGrade(percentile float64) string {
    switch {
	case percentile >= 90:
		return "A+"
    case percentile >= 80:
        return "A"
    case percentile >= 60:
        return "B"
    case percentile >= 40:
        return "C"
    case percentile >= 20:
        return "D"
	case percentile >= 10:
		return "E"
	default:
		return "F"
    }
}

type GradeThresholds struct {
    A_plus float64
    A float64
    B float64
    C float64
    D float64
    F float64
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
		fmt.Printf("gocloc fail. error: %v\n", err)
		return -1
	}

	for _, lang := range result.Languages {
		if lang.Name == language {
			return int(lang.Code)
		}
	}
	return 0
}

type SemgrepResultsFile struct {
	Results []interface{} `json:"results"`
}