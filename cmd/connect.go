package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
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

		// Only run setup if account metadata missing or reset requested
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

			fmt.Printf("Account '%s' saved\n", accountName)
		}

		acctCfg := cfg.Accounts[accountName]
		envName := strings.ToUpper(accountName)
		// Populate secret in the AccountConfig from environment
		switch acctCfg.AuthMethod {
		case "connection-string":
			acctCfg.Connection = os.Getenv(fmt.Sprintf("%s_CONNECTION_STRING", envName))
			if acctCfg.Connection == "" {
				return fmt.Errorf("missing environment variable: %s_CONNECTION_STRING", envName)
			}
		case "shared-key":
			acctCfg.AccountKey = os.Getenv(fmt.Sprintf("%s_ACCOUNT_KEY", envName))
			if acctCfg.AccountKey == "" {
				return fmt.Errorf("missing environment variable: %s_ACCOUNT_KEY", envName)
			}
		case "sas":
			acctCfg.ServiceURL = os.Getenv(fmt.Sprintf("%s_SAS_URL", envName))
			if acctCfg.ServiceURL == "" {
				return fmt.Errorf("missing environment variable: %s_SAS_URL", envName)
			}
		case "az-login":
			// no action needed, will use Azure CLI credential internally
		default:
			return fmt.Errorf("unsupported auth method: %s", acctCfg.AuthMethod)
		}

		client, err := azure.NewClientFromConfigAccount(acctCfg)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		fmt.Println("Testing connection...")
		if err := azure.TestConnection(client); err != nil {
			return fmt.Errorf("connection test failed: %w", err)
		}

		fmt.Printf("Successfully connected to account '%s'\n", accountName)
		return nil
	},
}

func setupAzLogin(name string) (*config.AccountConfig, error) {
	defaultURL := fmt.Sprintf("https://%s.blob.core.windows.net", name)
	acct := &config.AccountConfig{
		AccountName: name,
		ServiceURL:  defaultURL,
		AuthMethod:  "az-login",
	}
	fmt.Printf("Using Azure CLI credentials for account '%s' with Service URL '%s'\n", name, defaultURL)
	return acct, nil
}

func interactiveSetup(name string) (*config.AccountConfig, error) {
	acct := &config.AccountConfig{AccountName: name}
	authOptions := []string{"az-login", "connection-string", "shared-key", "sas"}

	survey.AskOne(&survey.Select{
		Message: "Choose auth method:",
		Options: authOptions,
	}, &acct.AuthMethod)

	// Service URL is always set based on account name
	acct.ServiceURL = fmt.Sprintf("https://%s.blob.core.windows.net", name)
	return acct, nil
}

func init() {
	connectCmd.Flags().BoolVar(&resetConfig, "reset", false, "Reset metadata for this account")
	connectCmd.Flags().BoolVar(&useAzLogin, "use-az-login", false, "Use Azure CLI login credentials")
}
