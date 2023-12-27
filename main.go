package main

import (
	"fmt"
	"git_pr_maker/cmd"
	local_helpers "git_pr_maker/pkg/local"
	"log"

	"github.com/spf13/cobra"
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

		local_helpers.GenerateOpsLevelTemplate(output, config)
	}

	//output = local_helpers.ReplaceTemplateValues(output, map[string]string{
	//	"test": "test",
	//})

	//fmt.Println(output)

}
