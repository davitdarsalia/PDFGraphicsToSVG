package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go /path/to/directory")
		os.Exit(1)
	}

	dirPath := os.Args[1]
	if !dirExists(dirPath) {
		fmt.Printf("Error: Directory not found: %s\n", dirPath)
		os.Exit(1)
	}

	fileList, err := listPDFFiles(dirPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	numWorkers := len(fileList)
	if numWorkers == 0 {
		fmt.Println("No PDF files found in the directory.")
		os.Exit(1)
	}

	processPDFsConcurrently(fileList, numWorkers)

	fmt.Println("All PDF files have been converted to SVG.")
}

func dirExists(dirPath string) bool {
	info, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func listPDFFiles(dirPath string) ([]string, error) {
	var pdfFiles []string
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".pdf") {
			pdfFiles = append(pdfFiles, filepath.Join(dirPath, file.Name()))
		}
	}

	return pdfFiles, nil
}

func processPDFsConcurrently(pdfFiles []string, numWorkers int) {
	jobs := make(chan string, len(pdfFiles))
	results := make(chan string, len(pdfFiles))

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go pdfToSVGWorker(jobs, results, &wg)
	}

	// Add jobs to the queue
	for _, file := range pdfFiles {
		jobs <- file
	}
	close(jobs)

	// Wait for workers to finish and close the results channel
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results
	for svgPath := range results {
		fmt.Printf("Successfully converted %s to %s\n", strings.TrimSuffix(svgPath, ".svg"), svgPath)
	}
}

func pdfToSVGWorker(jobs <-chan string, results chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()

	for pdfPath := range jobs {
		if !fileExists(pdfPath) {
			fmt.Printf("Error: File not found: %s\n", pdfPath)
			continue
		}

		svgPath := convertPDFToSVG(pdfPath)
		if svgPath == "" {
			fmt.Printf("Error: Conversion failed for %s\n", pdfPath)
			continue
		}

		if err := os.Remove(pdfPath); err != nil {
			fmt.Printf("Error: Failed to remove PDF file %s: %s\n", pdfPath, err)
			continue
		}

		results <- svgPath
	}
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func convertPDFToSVG(pdfPath string) string {
	svgPath := strings.TrimSuffix(pdfPath, filepath.Ext(pdfPath)) + ".svg"
	cmd := exec.Command("pdf2svg", pdfPath, svgPath)
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}
	return svgPath
}
