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
	debugMode      bool
	migrationDir   string
	compareBranch  string
	integration    bool
)

const (
	gitLabURLEnv          = "GITLAB_URL"
	gitLabTokenEnv        = "GITLAB_TOKEN"
	ciProjectIDEnv        = "CI_PROJECT_ID"
	ciMergeRequestIIDEnv  = "CI_MERGE_REQUEST_IID"
)

func init() {
	flag.StringVar(&migrationDir, "migration-dir", ".", "Specify the migration directory (default: current directory)")
	flag.StringVar(&compareBranch, "branch", "", "Specify the branch to compare against (default: empty, i.e., working directory)")
	flag.BoolVar(&debugMode, "debug", false, "Enable debug mode")
	flag.BoolVar(&integration, "integration", false, "Enable integration with GitLab merge request interface")

	flag.Parse()
}

func main() {
	if debugMode {
		fmt.Println("Debug mode enabled")
		fmt.Println("Migration directory: ", migrationDir)
		fmt.Println("Compare branch: ", compareBranch)
		fmt.Println("Integration enabled: ", integration)
	}

	// Read GitLab-related parameters from environment variables
	gitLabURL := os.Getenv(gitLabURLEnv)
	gitLabToken := os.Getenv(gitLabTokenEnv)
	ciProjectID := os.Getenv(ciProjectIDEnv)
	ciMergeRequestIID := os.Getenv(ciMergeRequestIIDEnv)

	if integration && ( gitLabURL == "" || gitLabToken == "" || ciProjectID == "" || ciMergeRequestIID == "" ){
		fmt.Println("Warning: GitLab-related environment variables not set.")
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

	cmdArgs = append(cmdArgs, migrationDir)

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

		if integration {
			if err := integrateWithGitLab(gitLabURL, gitLabToken, ciProjectID, ciMergeRequestIID, validationResult); err != nil {
				fmt.Println("Warning: Error integrating with GitLab:", err)
			}
		}
		os.Exit(1)
	}

	fmt.Println("Validation successful")
}

func integrateWithGitLab(gitLabURL, gitLabToken, ciProjectID, ciMergeRequestIID, validationResult string) error {
	// Prepare the curl command to post a note to the merge request with the validation result
	curlCommand := fmt.Sprintf(`curl --location --request POST "%s/api/v4/projects/%s/merge_requests/%s/notes" --header "PRIVATE-TOKEN: %s" --header "Content-Type: application/json" --data-raw "{ \"body\": \"%s" }"`, gitLabURL, ciProjectID, ciMergeRequestIID, gitLabToken, validationResult)

	cmd := exec.Command("bash", "-c", curlCommand)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error running curl command:", err)
		return err
	}

	// Log the output of the curl command (you can customize this as needed)
	fmt.Println("GitLab integration output:", string(output))

	return nil
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
