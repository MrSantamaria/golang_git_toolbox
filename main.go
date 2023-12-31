package main

import (
	"context"
	"fmt"
	"git_pr_maker/cmd"
	local_helpers "git_pr_maker/pkg/local"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

var dryRun bool // Flag to indicate whether it's a dry run

var rootCmd = &cobra.Command{
	Use:   "github",
	Short: "github is a personal tool used to automate GitHub tasks",
	Long:  `github is a personal tool used to automate GitHub tasks`,
	Run: func(cmd *cobra.Command, args []string) {
		//var errs []error
		//var err error

	},
}

func main() {
	cmd.InitEnv(rootCmd)

	// Your GitHub personal access token
	accessToken := "" // Replace with your actual access token
	templateyamlpath := "tmp/testfile.yml"
	repotxtpath := "tmp/repos.txt"
	dryRun = viper.GetBool("dry-run")
	var createdPRs []string

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	configs, err := local_helpers.ParseConfigFile(repotxtpath)
	if err != nil {
		log.Fatal(err)
	}

	for _, config := range configs {
		localRepoPath, err := cloneRepo(config.Repo, accessToken)
		if err != nil {
			log.Printf("Error cloning repository %s: %v\n", config.Repo, err)
			continue
		}

		defer os.RemoveAll(localRepoPath) // Clean up after processing

		output, err := local_helpers.ReadFile(templateyamlpath)
		if err != nil {
			log.Fatal(err)
		}

		filePath := filepath.Join(localRepoPath, "config.yaml")
		template := local_helpers.GenerateOpsLevelTemplate(output, config)

		err = writeToFile(filePath, template)
		if err != nil {
			log.Printf("Error writing to file %s: %v\n", filePath, err)
			continue
		}

		branchName, err := commitAndPushChanges(localRepoPath, config, accessToken) // Now returns the branch name
		if err != nil {
			log.Printf("Error committing and pushing changes for repository %s: %v\n", config.Repo, err)
			continue
		}

		owner, repo := extractOwnerAndRepo(config.Repo)
		err = createPullRequest(ctx, client, owner, repo, branchName, filePath, config.Repo, dryRun, &createdPRs)
		if err != nil {
			log.Printf("Error creating pull request: %v\n", err)
		}
	}

	// Print the list of created pull requests and their count
	fmt.Printf("\n%d Pull Request(s) created:\n", len(createdPRs))
	for _, prURL := range createdPRs {
		fmt.Println(prURL)
	}
}

func writeToFile(filePath, content string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

func createPullRequest(ctx context.Context, client *github.Client, owner, repo, branchName, filePath, repoURL string, dryRun bool, createdPRs *[]string) error {

	// No need to create a new branch reference, as it's already done in commitAndPushChanges

	if dryRun {
		fmt.Println("Dry-run mode: Pull request not created.")
		return nil
	}

	// Create a new pull request using the existing branch
	pr, _, err := client.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
		Title: github.String("Update configuration"),
		Head:  github.String(branchName), // Use the existing branch name
		Base:  github.String("main"),
		Body:  github.String("Automated configuration update"),
	})
	if err != nil {
		return err
	}

	prURL := pr.GetHTMLURL()
	*createdPRs = append(*createdPRs, prURL)
	fmt.Printf("Pull request created: %s\n", prURL)
	return nil
}

// Clone the repository and return the path to the local clone
func cloneRepo(repoURL, accessToken string) (string, error) {
	// Define the local path for the cloned repo
	tempDir, err := ioutil.TempDir("", "repo-clone-*")
	if err != nil {
		return "", err
	}

	// Construct the git clone command
	cmd := exec.Command("git", "clone", repoURL, tempDir)
	// Add other necessary setup for the command, such as setting environment variables for access tokens

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return tempDir, nil
}

func commitAndPushChanges(localRepoPath string, config local_helpers.RepositoryConfig, accessToken string) (string, error) {
	// Change directory to the local repository path
	if err := os.Chdir(localRepoPath); err != nil {
		return "", err
	}

	// Define a unique branch name
	branchName := fmt.Sprintf("update-config-%d", time.Now().UnixNano())

	// Check if the branch already exists
	checkBranchCmd := exec.Command("git", "rev-parse", "--verify", branchName)
	if err := checkBranchCmd.Run(); err != nil { // Branch does not exist
		// Create and switch to a new branch
		createBranchCmd := exec.Command("git", "checkout", "-b", branchName)
		if output, err := createBranchCmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("git checkout failed: %v, output: %s", err, string(output))
		}
	} else {
		// Switch to the existing branch
		switchBranchCmd := exec.Command("git", "checkout", branchName)
		if output, err := switchBranchCmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("git checkout failed: %v, output: %s", err, string(output))
		}
	}

	// Add, commit, and push changes
	addCmd := exec.Command("git", "add", "config.yaml")
	if output, err := addCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git add failed: %v, output: %s", err, string(output))
	}

	commitCmd := exec.Command("git", "commit", "-m", "Update configuration")
	if output, err := commitCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git commit failed: %v, output: %s", err, string(output))
	}

	pushCmd := exec.Command("git", "push", "-u", "origin", branchName)
	if output, err := pushCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git push failed: %v, output: %s", err, string(output))
	}

	return branchName, nil
}

// extractOwnerAndRepo extracts the owner and repo name from a GitHub repository URL.
func extractOwnerAndRepo(repoURL string) (string, string) {
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		// Handle the error as appropriate
		return "", ""
	}

	// Split the path into segments
	pathSegments := strings.Split(parsedURL.Path, "/")

	// Ensure there are at least two segments for owner and repo
	if len(pathSegments) < 3 {
		// Handle the error or invalid URL format
		return "", ""
	}

	// The owner and repo are usually the first and second segments of the path
	owner := pathSegments[1]
	repo := pathSegments[2]

	// Remove any .git extension from the repo name
	repo = strings.TrimSuffix(repo, ".git")

	return owner, repo
}
