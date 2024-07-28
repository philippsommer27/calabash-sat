package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type TestInfo struct {
    RuleName string
    Severity int
}

type CombinedResult struct {
    ProjectName    string
    IndividualGrades map[string]string
    OverallGrade   string
    AverageScore   float64
}

func CombineResults(resultsDir string, testInfos []TestInfo) error {
    projectResults := make(map[string]*CombinedResult)

    // First pass: collect all unique project names
    for _, testInfo := range testInfos {
        fileName := filepath.Join(resultsDir, fmt.Sprintf("*%s.json", testInfo.RuleName))
        matches, err := filepath.Glob(fileName)
        if err != nil || len(matches) == 0 {
            return fmt.Errorf("no results file found for rule %s", testInfo.RuleName)
        }

        resultsFile := matches[0]
        results, err := readResultsFile(resultsFile)
        if err != nil {
            return err
        }

        for _, project := range results.ProjectFindings {
            if _, exists := projectResults[project.ProjectName]; !exists {
                projectResults[project.ProjectName] = &CombinedResult{
                    ProjectName:     project.ProjectName,
                    IndividualGrades: make(map[string]string),
                }
            }
        }
    }

    // Second pass: fill in grades for each project and rule
    for _, testInfo := range testInfos {
        fileName := filepath.Join(resultsDir, fmt.Sprintf("*%s.json", testInfo.RuleName))
        matches, _ := filepath.Glob(fileName)
        resultsFile := matches[0]
        results, _ := readResultsFile(resultsFile)

        for projectName := range projectResults {
            grade := "A+" // Default grade if project not found in this file
            for _, project := range results.ProjectFindings {
                if project.ProjectName == projectName {
                    grade = project.Grade
                    break
                }
            }
            projectResults[projectName].IndividualGrades[testInfo.RuleName] = grade
        }
    }

    var combinedResults []CombinedResult
    for _, result := range projectResults {
        result.OverallGrade, result.AverageScore = calculateOverallGrade(result.IndividualGrades, testInfos)
        combinedResults = append(combinedResults, *result)
    }

    sort.Slice(combinedResults, func(i, j int) bool {
        return combinedResults[i].AverageScore > combinedResults[j].AverageScore
    })

    outputFile := filepath.Join(resultsDir, "combined_results.json")
    return writeJSONFile(outputFile, combinedResults)
}

func readResultsFile(filePath string) (EvalRulesResults, error) {
    var results EvalRulesResults
    data, err := os.ReadFile(filePath)
    if err != nil {
        return results, err
    }
    err = json.Unmarshal(data, &results)
    return results, err
}

func calculateOverallGrade(grades map[string]string, testInfos []TestInfo) (string, float64) {
    totalScore := 0.0
    totalWeight := 0
    for _, testInfo := range testInfos {
        grade, exists := grades[testInfo.RuleName]
        if !exists {
            // If no findings (no file), assign A+ (score of 6)
            totalScore += float64(6 * testInfo.Severity)
        } else {
            score := gradeToScore(grade)
            totalScore += float64(score * testInfo.Severity)
        }
        totalWeight += testInfo.Severity
    }

    if totalWeight == 0 {
        return "N/A", 0
    }

    averageScore := totalScore / float64(totalWeight)
    return scoreToGrade(averageScore), averageScore
}

func gradeToScore(grade string) int {
    switch grade {
    case "A+":
        return 6
    case "A":
        return 5
    case "B":
        return 4
    case "C":
        return 3
    case "D":
        return 2
    case "E":
        return 1
    case "F":
        return 0
    default:
        return -1
    }
}

func scoreToGrade(score float64) string {
    switch {
    case score >= 5.5:
        return "A+"
    case score >= 4.5:
        return "A"
    case score >= 3.5:
        return "B"
    case score >= 2.5:
        return "C"
    case score >= 1.5:
        return "D"
    case score >= 0.5:
        return "E"
    default:
        return "F"
    }
}

func writeJSONFile(filePath string, data interface{}) error {
    file, err := os.Create(filePath)
    if err != nil {
        return err
    }
    defer file.Close()

    encoder := json.NewEncoder(file)
    encoder.SetIndent("", "  ")
    return encoder.Encode(data)
}