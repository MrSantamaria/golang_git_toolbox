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
	"path"
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
	accessToken := ""

	// Create a GitHub client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	templateyamlpath := "tmp/testfile.yml"
	repotxtpath := "tmp/repos.txt"

	configs, err := local_helpers.ParseConfigFile(repotxtpath)
	if err != nil {
		log.Fatal(err)
	}

	// Parse the dry-run flag
	dryRun = viper.GetBool("dry-run")

	// Keep track of the created pull requests
	var createdPRs []string

	// Use the parsed configuration data as needed
	for _, config := range configs {
		// Extract the owner and repo from the repository URL
		repoURL, err := url.Parse(config.Repo)
		if err != nil {
			log.Printf("Error parsing repository URL %s: %v\n", config.Repo, err)
			continue
		}
		pathSegments := strings.Split(repoURL.Path, "/")
		fmt.Printf("Repo: %s, Tier: %d\n", config.Repo, config.Tier)

		output, err := local_helpers.ReadFile(templateyamlpath)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println(output)

		if len(pathSegments) < 3 {
			log.Printf("Invalid repository URL %s\n", config.Repo)
			continue
		}

		owner := pathSegments[1] // Assuming the second segment is the owner
		repo := pathSegments[2]  // Assuming the third segment is the repository name

		// Use ioutil.TempFile to create a temporary file with a recognizable prefix
		tempFile, err := ioutil.TempFile("", "config_*.yaml")
		if err != nil {
			log.Printf("Error creating temporary file: %v\n", err)
			continue
		}

		defer func() {
			// Close and delete the temporary file when done, log any errors
			tempFile.Close()
			if err := os.Remove(tempFile.Name()); err != nil {
				log.Printf("Error deleting temporary file %s: %v\n", tempFile.Name(), err)
			}
		}()

		template := local_helpers.GenerateOpsLevelTemplate(output, config)

		err = writeToFile(tempFile.Name(), template)
		if err != nil {
			log.Printf("Error writing to file %s: %v\n", tempFile.Name(), err)
			continue
		}

		// Create a pull request (passing dry-run flag)
		err = createPullRequest(ctx, client, owner, repo, tempFile.Name(), config.Repo, dryRun, &createdPRs)
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

func createPullRequest(ctx context.Context, client *github.Client, owner, repo, filePath, repoURL string, dryRun bool, createdPRs *[]string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Get the SHA of the default branch (e.g., main)
	baseRef, _, err := client.Repositories.GetBranch(ctx, owner, repo, "main")
	if err != nil {
		return err
	}

	// Create a new branch
	ref := fmt.Sprintf("refs/heads/update-config-%d", time.Now().UnixNano())
	_, _, err = client.Git.CreateRef(ctx, owner, repo, &github.Reference{
		Ref: &ref,
		Object: &github.GitObject{
			SHA: github.String(baseRef.Commit.GetSHA()), // Fix here
		},
	})
	if err != nil {
		return err
	}

	// Create a new commit
	tree, _, err := client.Git.CreateTree(ctx, owner, repo, baseRef.Commit.GetSHA(), []github.TreeEntry{
		{
			Path:    github.String(path.Base(filePath)),
			Mode:    github.String("100644"),
			Type:    github.String("blob"),
			Content: github.String(string(content)),
		},
	})
	if err != nil {
		return err
	}

	commit, _, err := client.Git.CreateCommit(ctx, owner, repo, &github.Commit{
		Message: github.String("Update configuration"),
		Tree:    tree,
		Parents: []github.Commit{
			{
				SHA: github.String(baseRef.Commit.GetSHA()), // Fix here
			},
		},
	})
	if err != nil {
		return err
	}

	// Create a new branch reference for the commit
	_, _, err = client.Git.UpdateRef(ctx, owner, repo, &github.Reference{
		Ref: github.String(ref),
		Object: &github.GitObject{
			SHA: commit.SHA,
		},
	}, false)
	if err != nil {
		return err
	}

	if dryRun {
		fmt.Println("Dry-run mode: Pull request not created.")
		return nil
	}

	// Create a new pull request
	pr, _, err := client.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
		Title: github.String("Update configuration"),
		Head:  github.String(ref),
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
