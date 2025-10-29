package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	rootCmd *cobra.Command
	version string
)

func Execute(ver string) {
	version = ver

	rootCmd = &cobra.Command{
		Use:   "azbutils",
		Short: "gsutil-like CLI for Azure Blob Storage",
	}

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(cpCmd)
	rootCmd.AddCommand(catCmd)
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(accountCmd)
	rootCmd.AddCommand(completionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
