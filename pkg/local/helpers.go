package local_helpers

import (
	"encoding/csv"
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
)

// RepositoryConfig holds configuration data for repositories.
type RepositoryConfig struct {
	Repo string
	Tier int
}

// ReadFile reads the content of a file and returns it as a string.
func ReadFile(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// ParseConfigFile reads a CSV-like file and returns a slice of RepositoryConfig.
func ParseConfigFile(filePath string) ([]RepositoryConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	lines, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var configs []RepositoryConfig
	for _, line := range lines {
		if len(line) != 2 {
			// Skip invalid lines
			continue
		}

		repo := line[0]
		tier, err := parseTier(line[1])
		if err != nil {
			// Skip lines with invalid tier
			continue
		}

		config := RepositoryConfig{
			Repo: repo,
			Tier: tier,
		}
		configs = append(configs, config)
	}

	return configs, nil

}

// parseTier converts a string to an integer, returning an error if the conversion fails.
func parseTier(s string) (int, error) {
	tier, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("failed to parse tier: %v", err)
	}
	return tier, nil
}

// ReplaceTemplateValues replaces template values in a string with the provided values.
func ReplaceTemplateValues(input string, values map[string]string) string {
	for key, value := range values {
		placeholder := "$" + key
		input = strings.ReplaceAll(input, placeholder, value)
	}
	return input
}

func GenerateOpsLevelTemplate(repoConfigOutput string, repoConfig RepositoryConfig) string {
	// Parse the repository URL
	repoURL, err := url.Parse(repoConfig.Repo)
	if err != nil {
		// Handle the error as needed
		return ""
	}
	// Extract the org and repository name
	org := strings.TrimPrefix(repoURL.Hostname(), "www.")
	repoName := path.Base(repoURL.Path)

	// Extract the repository name without the org and without underscores for description
	repoDescription := strings.ReplaceAll(repoName, "_", " ")
	// TODO: Replace this logic with a pointer
	copyofrepoConfigOutput := repoConfigOutput

	copyofrepoConfigOutput = strings.ReplaceAll(copyofrepoConfigOutput, "${REPO_TIER}", fmt.Sprintf("tier_%d", repoConfig.Tier))
	copyofrepoConfigOutput = strings.ReplaceAll(copyofrepoConfigOutput, "${REPO_NAME}", repoName)
	copyofrepoConfigOutput = strings.ReplaceAll(copyofrepoConfigOutput, "${REPO_FULL_NAME}", path.Join(org, repoName))
	copyofrepoConfigOutput = strings.ReplaceAll(copyofrepoConfigOutput, "${REPO_DESCRIPTION}", repoDescription)

	fmt.Println(copyofrepoConfigOutput)

	return copyofrepoConfigOutput
}
