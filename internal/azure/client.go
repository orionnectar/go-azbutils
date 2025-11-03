package azure

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/orionnectar/go-azbutils/internal/config"
)

func NewClientFromConfigAccount(acct *config.AccountConfig) (*azblob.Client, error) {
	envName := strings.ToUpper(acct.AccountName)
	switch acct.AuthMethod {
	case "connection-string":
		cs := os.Getenv(fmt.Sprintf("%s_CONNECTION_STRING", envName))
		if cs == "" {
			return nil, fmt.Errorf("Missing environment variable: %s_CONNECTION_STRING", envName)
		}
		return azblob.NewClientFromConnectionString(cs, nil)
	case "shared-key":
		key := os.Getenv(fmt.Sprintf("%s_ACCOUNT_KEY", envName))
		if key == "" {
			return nil, fmt.Errorf("Missing environment variable: %s_ACCOUNT_KEY", envName)
		}
		cred, err := azblob.NewSharedKeyCredential(acct.AccountName, key)
		if err != nil {
			return nil, err
		}
		return azblob.NewClientWithSharedKeyCredential(acct.ServiceURL, cred, nil)
	case "sas":
		sas := os.Getenv(fmt.Sprintf("%s_SAS_URL", envName))
		if sas == "" {
			return nil, fmt.Errorf("Missing environment variable: %s_SAS_URL", envName)
		}
		return azblob.NewClientWithNoCredential(sas, nil)
	case "az-login", "default":
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, err
		}
		return azblob.NewClient(acct.ServiceURL, cred, nil)
	default:
		return nil, fmt.Errorf("Unsupported auth method: %s", acct.AuthMethod)
	}
}

func TestConnection(client *azblob.Client) error {
	ctx := context.Background()
	pager := client.NewListContainersPager(nil)

	if pager.More() {
		_, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("Failed to list containers: %w", err)
		}
	}
	return nil
}
