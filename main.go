package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func main() {
	// Parse command-line arguments
	migrationDir := "."
	for i, arg := range os.Args {
		if arg == "--migration-dir" && i+1 < len(os.Args) {
			migrationDir = os.Args[i+1]
			break
		}
	}

	// Run git diff command
	cmd := exec.Command("git", "diff", "--name-status")
	cmd.Dir = migrationDir
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error running git diff:", err)
		os.Exit(1)
	}

	// Parse and check conditions for each file
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) == 3 {
			status := fields[0]
			oldFilePath := fields[1]
			newFilePath := fields[2]

			switch status {
			case "M":
				if isMigrationFile(newFilePath) && !isAlphabeticallyLast(newFilePath, migrationDir) {
					fmt.Println("Error: Modified migration file not alphabetically last:", newFilePath)
					os.Exit(1)
				}
			case "A":
				if isMigrationFile(newFilePath) && !isAlphabeticallyLast(newFilePath, migrationDir) {
					fmt.Println("Error: Added migration file not alphabetically last:", newFilePath)
					os.Exit(1)
				}
			case "D":
				if isMigrationFile(oldFilePath) {
					fmt.Println("Error: Removed migration file:", oldFilePath)
					os.Exit(1)
				}
			case "R":
				if isMigrationFile(newFilePath) && !isAlphabeticallyLast(newFilePath, migrationDir) {
					fmt.Println("Error: Renamed migration file not alphabetically last:", newFilePath)
					os.Exit(1)
				}
			}
		}
	}

	fmt.Println("Validation successful")
}

func isMigrationFile(filePath string) bool {
	re := regexp.MustCompile(`^V\d+__$`)
	return re.MatchString(filepath.Base(filePath))
}

func isAlphabeticallyLast(filePath, migrationDir string) bool {
	files, err := getMigrationFiles(migrationDir)
	if err != nil {
		fmt.Println("Error getting migration files:", err)
		os.Exit(1)
	}

	sort.Strings(files)

	for i, file := range files {
		if file == filePath {
			return i == len(files)-1
		}
	}

	return false
}

func getMigrationFiles(migrationDir string) ([]string, error) {
	var files []string

	err := filepath.Walk(migrationDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isMigrationFile(path) {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}
