package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	debugMode     bool
	migrationDir  string
	compareBranch string
)

func init() {
	flag.StringVar(&migrationDir, "migration-dir", ".", "Specify the migration directory (default: current directory)")
	flag.StringVar(&compareBranch, "branch", "", "Specify the branch to compare against (default: empty, i.e., working directory)")
	flag.BoolVar(&debugMode, "debug", false, "Enable debug mode")

	flag.Parse()
}

func main() {
	if flag.NArg() > 0 && (flag.Arg(0) == "--help" || flag.Arg(0) == "-h") {
		printHelp()
		os.Exit(0)
	}

	if debugMode {
		fmt.Println("Debug mode enabled")
		fmt.Println("Migration directory: ", migrationDir)
		fmt.Println("Compare branch: ", compareBranch)
		dir, err := os.Getwd()
		if err != nil {
			fmt.Println("Err", err)
		}
		fmt.Println(dir)
	}

	if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
		fmt.Println("Error: Migration directory does not exist:", migrationDir)
		os.Exit(1)
	}

	if debugMode {
		fmt.Printf("Verify dir :%s: \n", migrationDir)
	}

	cmdArgs := []string{"git", "diff", "--name-status"}

	if compareBranch != "" {
		cmdArgs = append(cmdArgs, compareBranch)
	}

	cmdArgs = append(cmdArgs, "--", migrationDir)

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error running git diff:", err)
		if debugMode {
			fmt.Println(string(cmd.String()))
		}
		os.Exit(1)
	}

	var validationErrors []string

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) >= 2 {
			status := fields[0]
			oldFilePath := fields[1]
			newFilePath := oldFilePath

			if len(fields) == 3 {
				newFilePath = fields[2]
			}

			relativeFilePath, err := filepath.Rel(migrationDir, newFilePath)
			if err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("Error getting relative path: %v", err))
				continue
			}

			if debugMode {
				fmt.Printf("Processing file: %s (status: %s)\n", newFilePath, status)
				fmt.Printf("Conditions: isMigrationFile: %t, isAlphabeticallyLast: %t\n", isMigrationFile(relativeFilePath), isAlphabeticallyLast(relativeFilePath, migrationDir))
			}

			// Regex for <X><score>
			if isMatch(status, `^M`) {
				if isMigrationFile(relativeFilePath) {
					validationErrors = append(validationErrors, fmt.Sprintf("Error: Cannot modify migration file after it was applied: %s\n\t%s", relativeFilePath, newFilePath))
				}
			} else if isMatch(status, `^A`) {
				if isMigrationFile(relativeFilePath) && !isAlphabeticallyLast(relativeFilePath, migrationDir) {
					validationErrors = append(validationErrors, fmt.Sprintf("Error: Added migration file not alphabetically last: %s\n\t%s", relativeFilePath, newFilePath))
				}
			} else if isMatch(status, `^D`) {
				if isMigrationFile(relativeFilePath) {
					validationErrors = append(validationErrors, fmt.Sprintf("Error: Cannot remove migration file after it was applied: %s\n\t%s", relativeFilePath, newFilePath))
				}
			} else if isMatch(status, `^R`) {
				if isMigrationFile(relativeFilePath) {
					validationErrors = append(validationErrors, fmt.Sprintf("Error: Cannot rename migration file after it was applied: %s\n\t%s", relativeFilePath, newFilePath))
				}
			}
		}
	}

	if len(validationErrors) > 0 {
		fmt.Println("Validation errors:")
		for _, err := range validationErrors {
			fmt.Println(err)
		}
		os.Exit(1)
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
		return false
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

func isMatch(input, pattern string) bool {
	re := regexp.MustCompile(pattern)
	return re.MatchString(input)
}

func printHelp() {
	fmt.Println("Usage: dum-flyway-validate [OPTIONS]")
	fmt.Println("  --migration-dir   Specify the migration directory (default: current directory)")
	fmt.Println("  --debug            Enable debug mode")
	fmt.Println("  --help, -h         Show this help message")
}
