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

var debugMode bool

func main() {
	// Check for --help
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		printHelp()
		os.Exit(0)
	}

	// Parse command-line arguments
	migrationDir := "."
	for i, arg := range os.Args {
		if arg == "--migration-dir" && i+1 < len(os.Args) {
			migrationDir = os.Args[i+1]
			break
		} else if arg == "--debug" {
			debugMode = true
		}
	}

	if debugMode {
		fmt.Println("Debug mode enabled")
		fmt.Println("Migration directory:", migrationDir)
	}

	// Change the current working directory
	if err := os.Chdir(migrationDir); err != nil {
		fmt.Println("Error changing directory:", err)
		os.Exit(1)
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

		if len(fields) >= 2 {
			status := fields[0]
			oldFilePath := fields[1]
			newFilePath := oldFilePath // Assume no rename

			if len(fields) == 3 {
				newFilePath = fields[2]
			}

			// Skip files not in the specified directory
			if !strings.HasPrefix(newFilePath, migrationDir) {
				if debugMode {
					fmt.Printf("Skipping file: %s (status: %s)\n", newFilePath, status)
				}
				continue
			}

			// Remove migration-dir from the file path
			relativeFilePath, err := filepath.Rel(migrationDir, newFilePath)
			if err != nil {
				fmt.Println("Error getting relative path:", err)
				os.Exit(1)
			}

			if debugMode {
				fmt.Printf("Processing file: %s (status: %s)\n", newFilePath, status)
				fmt.Printf("Conditions: isMigrationFile: %t, isAlphabeticallyLast: %t\n", isMigrationFile(newFilePath), isAlphabeticallyLast(newFilePath, migrationDir))
			}

			switch status {
			case "M":
				if isMigrationFile(relativeFilePath) {
					fmt.Println("Error: Cannot modify migration file after it was applied:", relativeFilePath)
					os.Exit(1)
				}
			case "A":
				if isMigrationFile(relativeFilePath) && !isAlphabeticallyLast(relativeFilePath, migrationDir) {
					fmt.Println("Error: Added migration file not alphabetically last:", relativeFilePath)
					os.Exit(1)
				}
			case "D":
				if isMigrationFile(relativeFilePath) {
					fmt.Println("Error: Cannot remove migration file after it was applied:", relativeFilePath)
					os.Exit(1)
				}
			case "R":
				if isMigrationFile(relativeFilePath) {
					fmt.Println("Error: Cannot rename migration file after it was applied:", relativeFilePath)
					os.Exit(1)
				}
			}
		}
	}

	fmt.Println("Validation successful")
}

func isMigrationFile(filePath string) bool {
	re := regexp.MustCompile(`^V\d+__`)
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

func printHelp() {
	fmt.Println("Usage: dum-flyway-validate [OPTIONS]")
	fmt.Println("  --migration-dir   Specify the migration directory (default: current directory)")
	fmt.Println("  --debug            Enable debug mode")
	fmt.Println("  --help, -h         Show this help message")
}
