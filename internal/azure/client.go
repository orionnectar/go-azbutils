package azure

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/orionnectar/go-azbutils/internal/config"
)

// Build a new client based on config.json
func NewClientFromConfigAccount(acct *config.AccountConfig) (*azblob.Client, error) {
	switch acct.AuthMethod {
	case "connection-string":
		return azblob.NewClientFromConnectionString(acct.Connection, nil)
	case "shared-key":
		cred, err := azblob.NewSharedKeyCredential(acct.AccountName, acct.AccountKey)
		if err != nil {
			return nil, err
		}
		return azblob.NewClientWithSharedKeyCredential(acct.ServiceURL, cred, nil)
	case "sas":
		return azblob.NewClientWithNoCredential(acct.ServiceURL, nil)
	case "az-login", "default":
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, err
		}
		return azblob.NewClient(acct.ServiceURL, cred, nil)
	default:
		return nil, fmt.Errorf("unsupported auth method: %s", acct.AuthMethod)
	}
}

// NewClientWithCredential creates a new Azure Blob client using a provided azcore.TokenCredential (e.g. Azure CLI login)
func NewClientWithCredential(serviceURL string, cred azcore.TokenCredential) (*azblob.Client, error) {
	client, err := azblob.NewClient(serviceURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create client with credential: %w", err)
	}
	return client, nil
}

// TestConnection ensures that client is valid and has access.
func TestConnection(client *azblob.Client) error {
	ctx := context.Background()
	pager := client.NewListContainersPager(nil)

	if pager.More() {
		_, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}
	}
	return nil
}
