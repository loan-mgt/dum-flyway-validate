package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
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
	integration   bool
)

const (
	gitLabURLEnv         = "GITLAB_URL"
	gitLabTokenEnv       = "GITLAB_TOKEN"
	ciProjectIDEnv       = "CI_PROJECT_ID"
	ciMergeRequestIIDEnv = "CI_MERGE_REQUEST_IID"
)

type ValidationError struct {
	Message   string
	MessageMD string
	OldPath   string
	NewPath   string
}

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

	if integration && (gitLabURL == "" || gitLabToken == "" || ciProjectID == "" || ciMergeRequestIID == "") {
		fmt.Println("Warning: GitLab-related environment variables not set.")
	}

	if debugMode {
		fmt.Println("GitLab URL: ", gitLabURL)
		fmt.Println("GitLab Token: ", len(gitLabToken))
		fmt.Println("CI Project ID: ", ciProjectID)
		fmt.Println("CI Merge Request IID: ", ciMergeRequestIID)
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

	var validationErrors []ValidationError

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
				validationErrors = append(validationErrors, ValidationError{
					Message: fmt.Sprintf("Error getting relative path: %v", err),
				})
				continue
			}

			if debugMode {
				fmt.Printf("Processing file: %s (status: %s)\n", newFilePath, status)
				fmt.Printf("Conditions: isMigrationFile: %t, isAlphabeticallyLast: %t\n", isMigrationFile(relativeFilePath), isAlphabeticallyLast(relativeFilePath, migrationDir))
			}

			if isMatch(status, `^M`) {
				if isMigrationFile(relativeFilePath) {
					validationErrors = append(validationErrors, ValidationError{
						Message:   fmt.Sprintf("Error: Cannot modify migration file after it was applied"),
						MessageMD: fmt.Sprintf(":warning: Error: Cannot **modify** migration file after it was applied"),
						OldPath:   oldFilePath,
						NewPath:   newFilePath,
					})
				}
			} else if isMatch(status, `^A`) {
				if isMigrationFile(relativeFilePath) && !isAlphabeticallyLast(relativeFilePath, migrationDir) {
					validationErrors = append(validationErrors, ValidationError{
						Message:   fmt.Sprintf("Error: Added migration file not alphabetically last"),
						MessageMD: fmt.Sprintf(":warning: Error: Added migration file **not** alphabetically **last**"),
						OldPath:   oldFilePath,
						NewPath:   newFilePath,
					})
				}
			} else if isMatch(status, `^D`) {
				if isMigrationFile(relativeFilePath) {
					validationErrors = append(validationErrors, ValidationError{
						Message:   fmt.Sprintf("Error: Cannot remove migration file after it was applied"),
						MessageMD: fmt.Sprintf(":warning: Error: Cannot **remove** migration file after it was applied"),
						OldPath:   oldFilePath,
						NewPath:   newFilePath,
					})
				}
			} else if isMatch(status, `^R`) {
				if isMigrationFile(relativeFilePath) {
					validationErrors = append(validationErrors, ValidationError{
						Message:   fmt.Sprintf("Error: Cannot rename migration file after it was applied"),
						MessageMD: fmt.Sprintf(":warning: Error: Cannot **rename** migration file after it was applied"),
						OldPath:   oldFilePath,
						NewPath:   newFilePath,
					})
				}
			}
		}
	}

	if len(validationErrors) > 0 {
		fmt.Println("Validation errors:")
		for _, err := range validationErrors {
			fmt.Printf("%s\n	%s\n\n", err.Message, err.NewPath)
		}

		if integration {
			if err := integrateWithGitLab(gitLabURL, gitLabToken, ciProjectID, ciMergeRequestIID, validationErrors); err != nil {
				fmt.Println("Warning: Error integrating with GitLab:", err)
			}
		}
		os.Exit(1)
	}

	fmt.Println("Validation successful")
}

func RetrieveMergeRequestInfo(gitLabURL, gitLabToken, ciProjectID, ciMergeRequestIID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/v4/projects/%s/merge_requests/%s", gitLabURL, ciProjectID, ciMergeRequestIID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating HTTP request: %v", err)
	}

	req.Header.Set("PRIVATE-TOKEN", gitLabToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Error retrieving Merge Request info. Status code: %d", resp.StatusCode)
	}

	var mrInfo map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&mrInfo); err != nil {
		return nil, fmt.Errorf("Error decoding Merge Request info: %v", err)
	}

	return mrInfo, nil
}

func integrateWithGitLab(gitLabURL, gitLabToken, ciProjectID, ciMergeRequestIID string, validationErrors []ValidationError) error {
	mrInfo, err := RetrieveMergeRequestInfo(gitLabURL, gitLabToken, ciProjectID, ciMergeRequestIID)
	if err != nil {
		return fmt.Errorf("Error retrieving Merge Request info: %v", err)
	}

	for _, validationError := range validationErrors {
		// Prepare the URL to post a discussion to the merge request with the validation result
		url := fmt.Sprintf("%s/v4/projects/%s/merge_requests/%s/discussions", gitLabURL, ciProjectID, ciMergeRequestIID)

		// Create a map for the JSON payload
		payload := map[string]interface{}{
			"body": validationError.MessageMD,
			"position": map[string]interface{}{
				"position_type":            "file",
				"base_sha":                 mrInfo["diff_refs"].(map[string]interface{})["base_sha"],
				"head_sha":                 mrInfo["diff_refs"].(map[string]interface{})["head_sha"],
				"start_sha":                mrInfo["diff_refs"].(map[string]interface{})["start_sha"],
				"new_path":                 validationError.NewPath,
				"old_path":                 validationError.OldPath,
				"old_line":                 nil,
				"new_line":                 nil,
				"line_range":               map[string]interface{}{},
				"ignore_whitespace_change": false,
			},
		}

		// Marshal the payload into JSON
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("Error marshalling JSON payload: %v", err)
		}

		// Create a new HTTP request with the JSON payload
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
		if err != nil {
			return fmt.Errorf("Error creating HTTP request: %v", err)
		}

		// Set headers for authentication and content type
		req.Header.Set("PRIVATE-TOKEN", gitLabToken)
		req.Header.Set("Content-Type", "application/json")

		// Perform the HTTP request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("Error making HTTP request: %v", err)
		}
		defer resp.Body.Close()

		// Check the HTTP response status
		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("Error posting discussion to merge request. Status code: %d", resp.StatusCode)
		}
	}

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
