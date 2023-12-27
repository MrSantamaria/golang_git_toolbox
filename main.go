package main

import (
	"fmt"
	"os"

	"github.com/MrSantamaria/git_pr_maker/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "acceptance_test",
	Short: "acceptance_test is a component of the Hypershift Operator Promotion process",
	Long:  `acceptance_test is a tool used to validate Hypershift Operator Promotions ocurred successfully`,
	Run: func(cmd *cobra.Command, args []string) {
		var errs []error
		var err error

		err = workflows.SetUp(viper.GetString("token"), viper.GetString("environment"))
		if err != nil {
			fmt.Println(err)
			errs = append(errs, err)
		}

		err = workflows.AcceptanceTest()
		if err != nil {
			fmt.Println(err)
			errs = append(errs, err)
		}

		err = workflows.CleanUp()
		if err != nil {
			fmt.Println(err)
			errs = append(errs, err)
		}

		if len(errs) > 0 {
			fmt.Printf("Acceptance Test FAILED for: %s %s environment: %s selectors: %v\n",
				viper.GetString("operator"),
				viper.GetString("imagetag"),
				viper.GetString("environment"),
				viper.GetStringSlice("selectors"))
			os.Exit(1)
		}
	},
}

func main() {
	cmd.InitEnv(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	os.Exit(0)
}
