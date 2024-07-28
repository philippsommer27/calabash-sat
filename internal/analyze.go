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
	"sync"
	"sync/atomic"
)

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

type SemgrepResultsFile struct {
	Results []interface{} `json:"results"`
}