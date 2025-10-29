package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/orionnectar/go-azbutils/internal/azure"
	"github.com/orionnectar/go-azbutils/internal/config"
	"github.com/spf13/cobra"
)

var (
	resetConfig bool
	useAzLogin  bool
)

var connectCmd = &cobra.Command{
	Use:   "connect [account_name]",
	Short: "Connect to an Azure Storage account and verify connection",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := config.Load()
		if cfg == nil {
			cfg = &config.Config{Accounts: make(map[string]*config.AccountConfig)}
		}

		var accountName string
		if len(args) == 1 {
			accountName = args[0]
		} else {
			accountName = cfg.DefaultAccount
		}

		if accountName == "" {
			return fmt.Errorf("please specify an account name (e.g. azbutils connect myaccount)")
		}

		if resetConfig || cfg.Accounts[accountName] == nil {
			var acctCfg *config.AccountConfig
			var err error

			if useAzLogin {
				acctCfg, err = setupAzLogin(accountName)
			} else {
				acctCfg, err = interactiveSetup(accountName)
			}

			if err != nil {
				return err
			}
			cfg.Accounts[accountName] = acctCfg

			if cfg.DefaultAccount == "" {
				cfg.DefaultAccount = accountName
			}

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			fmt.Printf("💾 Account '%s' saved.\n", accountName)
		}

		acctCfg := cfg.Accounts[accountName]
		client, err := azure.NewClientFromConfigAccount(acctCfg)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		fmt.Println("🔗 Testing connection...")
		if err := azure.TestConnection(client); err != nil {
			return fmt.Errorf("connection test failed: %w", err)
		}

		fmt.Printf("✅ Successfully connected to account '%s'\n", accountName)
		return nil
	},
}

// setupAzLogin handles credential setup via Azure CLI login
func setupAzLogin(name string) (*config.AccountConfig, error) {
	fmt.Println("🔑 Using Azure CLI credentials...")

	// Test if Azure CLI credentials are available
	cred, err := azidentity.NewAzureCLICredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure CLI credential: %w", err)
	}

	var serviceURL string
	survey.AskOne(&survey.Input{
		Message: "Service URL (e.g. https://<account>.blob.core.windows.net):",
	}, &serviceURL)

	acct := &config.AccountConfig{
		AccountName: name,
		ServiceURL:  serviceURL,
		AuthMethod:  "az-login",
	}

	// Test connection directly
	client, err := azure.NewClientWithCredential(serviceURL, cred)
	if err != nil {
		return nil, fmt.Errorf("failed to test Azure CLI credential: %w", err)
	}

	if err := azure.TestConnection(client); err != nil {
		return nil, fmt.Errorf("Azure CLI credential test failed: %w", err)
	}

	fmt.Println("✅ Azure CLI credentials verified")
	return acct, nil
}

func interactiveSetup(name string) (*config.AccountConfig, error) {
	acct := &config.AccountConfig{AccountName: name}

	authOptions := []string{"az-login", "connection-string", "shared-key", "sas"}
	survey.AskOne(&survey.Select{
		Message: "Choose auth method:",
		Options: authOptions,
	}, &acct.AuthMethod)

	survey.AskOne(&survey.Input{
		Message: "Service URL (e.g. https://account.blob.core.windows.net):",
	}, &acct.ServiceURL)

	switch acct.AuthMethod {
	case "connection-string":
		survey.AskOne(&survey.Password{Message: "Connection string:"}, &acct.Connection)
	case "shared-key":
		survey.AskOne(&survey.Input{Message: "Account name:"}, &acct.AccountName)
		survey.AskOne(&survey.Password{Message: "Account key:"}, &acct.AccountKey)
	case "sas":
		survey.AskOne(&survey.Input{Message: "SAS URL (with ?sig=...):"}, &acct.ServiceURL)
	}

	return acct, nil
}

func init() {
	connectCmd.Flags().BoolVar(&resetConfig, "reset", false, "Reset configuration for this account")
	connectCmd.Flags().BoolVar(&useAzLogin, "use-az-login", false, "Use Azure CLI login credentials")
}
