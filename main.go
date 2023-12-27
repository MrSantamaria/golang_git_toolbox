package main

import (
	"context"
	"fmt"
	"git_pr_maker/cmd"
	local_helpers "git_pr_maker/pkg/local"
	"log"
	"os"
	"path"

	"github.com/google/go-github/github"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

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
	accessToken := "your-access-token"

	// Create a GitHub client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Your GitHub username and repository
	owner := "your-username"
	repo := "your-repository"

	templateyamlpath := "tmp/testfile.yml"
	repotxtpath := "tmp/repos.txt"

	configs, err := local_helpers.ParseConfigFile(repotxtpath)
	if err != nil {
		log.Fatal(err)
	}

	// Use the parsed configuration data as needed
	for _, config := range configs {
		fmt.Printf("Repo: %s, Tier: %d\n", config.Repo, config.Tier)
		output, err := local_helpers.ReadFile(templateyamlpath)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println(output)

		template := local_helpers.GenerateOpsLevelTemplate(output, config)

		// Write the YAML content to a file
		outputFilePath := path.Join("path/to/output", fmt.Sprintf("%s.yaml", config.Repo))
		err = writeToFile(outputFilePath, template)
		if err != nil {
			log.Printf("Error writing to file %s: %v\n", outputFilePath, err)
			continue
		}

		// Create a pull request
		err = createPullRequest(ctx, client, owner, repo, outputFilePath, config.Repo)
		if err != nil {
			log.Printf("Error creating pull request: %v\n", err)
		}

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

func createPullRequest(ctx context.Context, client *github.Client, owner, repo, filePath, repoURL string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Create a new branch
	ref := "refs/heads/update-config"
	baseRef := "main"
	_, _, err = client.Git.CreateRef(ctx, owner, repo, &github.Reference{
		Ref: &ref,
		Object: &github.GitObject{
			SHA: github.String(baseRef),
		},
	})
	if err != nil {
		return err
	}

	// Create a new commit
	tree, _, err := client.Git.CreateTree(ctx, owner, repo, baseRef, []github.TreeEntry{
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
				SHA: github.String(baseRef),
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

	// Create a new pull request
	pr, _, err := client.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
		Title: github.String("Update configuration"),
		Head:  github.String(ref),
		Base:  github.String(baseRef),
		Body:  github.String("Automated configuration update"),
	})
	if err != nil {
		return err
	}

	fmt.Printf("Pull request created: %s\n", pr.GetHTMLURL())
	return nil
}
