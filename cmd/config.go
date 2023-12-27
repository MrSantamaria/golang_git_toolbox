package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	selectors []string
)

func InitEnv(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().String("file", "", "File Path")
	rootCmd.PersistentFlags().Bool("dry-run", true, "Dry Run")

	viper.BindPFlag("file", rootCmd.PersistentFlags().Lookup("file"))
	viper.BindPFlag("dry-run", rootCmd.PersistentFlags().Lookup("dry-run"))

	viper.AutomaticEnv()
}
