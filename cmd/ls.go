package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/orionnectar/go-azbutils/internal/azure"
	"github.com/orionnectar/go-azbutils/internal/config"
	"github.com/spf13/cobra"
)

var recursive bool

var lsCmd = &cobra.Command{
	Use:   "ls [az://account//container[/path]]",
	Short: "List blobs in a container or virtual directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := strings.TrimPrefix(args[0], "az://")
		parts := strings.SplitN(path, "//", 2)
		if len(parts) < 2 {
			return fmt.Errorf("invalid path format. use az://<account>//<container>")
		}

		accountName := parts[0]
		containerPath := parts[1]

		containerParts := strings.SplitN(containerPath, "/", 2)
		containerName := containerParts[0]
		prefix := ""
		if len(containerParts) == 2 {
			prefix = containerParts[1]
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		acctCfg := cfg.Accounts[accountName]
		if acctCfg == nil {
			return fmt.Errorf("no account config found for '%s'", accountName)
		}

		client, err := azure.NewClientFromConfigAccount(acctCfg)
		if err != nil {
			return fmt.Errorf("failed to create Azure client: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		containerClient := client.ServiceClient().NewContainerClient(containerName)

		fmt.Printf("Listing blobs in '%s' (account: %s):\n", containerName, accountName)

		if recursive {
			// Recursive: list all blobs (flat)
			pager := containerClient.NewListBlobsFlatPager(&azblob.ListBlobsFlatOptions{Prefix: &prefix})
			for pager.More() {
				page, err := pager.NextPage(ctx)
				if err != nil {
					return fmt.Errorf("list error: %w", err)
				}
				for _, blob := range page.Segment.BlobItems {
					fmt.Printf(" 📄 %s\n", *blob.Name)
				}
			}
		} else {
			// Non-recursive: list only immediate blobs + prefixes
			pager := containerClient.NewListBlobsHierarchyPager("/", &container.ListBlobsHierarchyOptions{Prefix: &prefix})
			for pager.More() {
				page, err := pager.NextPage(ctx)
				if err != nil {
					return fmt.Errorf("list error: %w", err)
				}
				for _, prefix := range page.Segment.BlobPrefixes {
					fmt.Printf(" 📁 %s\n", *prefix.Name)
				}
				for _, blob := range page.Segment.BlobItems {
					fmt.Printf(" 📄 %s\n", *blob.Name)
				}
			}
		}

		return nil
	},
}

func init() {
	lsCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively list all blobs")
}
