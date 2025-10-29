package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/orionnectar/go-azbutils/internal/config"
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage multiple Azure storage accounts",
}

var accountAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a new storage account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfg, _ := config.Load()
		if cfg == nil {
			cfg = &config.Config{Accounts: make(map[string]*config.AccountConfig)}
		}

		acct := &config.AccountConfig{}
		// Interactive survey
		authOptions := []string{"az-login", "connection-string", "shared-key", "sas"}
		survey.AskOne(&survey.Select{
			Message: "Choose auth method:",
			Options: authOptions,
		}, &acct.AuthMethod)

		survey.AskOne(&survey.Input{Message: "Service URL (e.g. https://account.blob.core.windows.net):"}, &acct.ServiceURL)

		switch acct.AuthMethod {
		case "connection-string":
			survey.AskOne(&survey.Password{Message: "Connection string:"}, &acct.Connection)
		case "shared-key":
			survey.AskOne(&survey.Input{Message: "Account name:"}, &acct.AccountName)
			survey.AskOne(&survey.Password{Message: "Account key:"}, &acct.AccountKey)
		case "sas":
			survey.AskOne(&survey.Input{Message: "SAS URL (with ?sig=...):"}, &acct.ServiceURL)
		}

		cfg.Accounts[name] = acct
		if cfg.DefaultAccount == "" {
			cfg.DefaultAccount = name
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("✅ Added account '%s' (default: %s)\n", name, cfg.DefaultAccount)
		return nil
	},
}

var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		fmt.Println("Accounts:")
		for name := range cfg.Accounts {
			marker := ""
			if name == cfg.DefaultAccount {
				marker = "(default)"
			}
			fmt.Println(" -", name, marker)
		}
		return nil
	},
}

var accountSetDefaultCmd = &cobra.Command{
	Use:   "default [name]",
	Short: "Set default account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if _, ok := cfg.Accounts[name]; !ok {
			return fmt.Errorf("account '%s' not found", name)
		}
		cfg.DefaultAccount = name
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("✅ Set '%s' as default account\n", name)
		return nil
	},
}

func init() {
	accountCmd.AddCommand(accountAddCmd)
	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountSetDefaultCmd)
}
