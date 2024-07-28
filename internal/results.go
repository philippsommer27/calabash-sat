package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

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

type GradeThresholds struct {
    A_plus float64
    A      float64
    B      float64
    C      float64
    D      float64
    F      float64
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