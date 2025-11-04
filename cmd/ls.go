package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/orionnectar/go-azbutils/internal/azpath"
	"github.com/orionnectar/go-azbutils/internal/azure"
	"github.com/orionnectar/go-azbutils/internal/config"
	"github.com/spf13/cobra"
)

var (
	recursive bool
	fullPath  bool
	pretty    bool
)

var lsCmd = &cobra.Command{
	Use:   "ls [az://account//container[/path]] or [https://...]",
	Short: "List blobs in a container or virtual directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := azpath.Parse(args[0])
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		acctCfg := cfg.Accounts[p.Account]
		if acctCfg == nil {
			return fmt.Errorf("no account config found for '%s'", p.Account)
		}

		client, err := azure.NewClientFromConfigAccount(acctCfg)
		if err != nil {
			return fmt.Errorf("failed to create Azure client: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		containerClient := client.ServiceClient().NewContainerClient(p.Container)

		fmt.Printf("Listing blobs in '%s' (account: %s):\n", p.Container, p.Account)

		if recursive {
			pager := containerClient.NewListBlobsFlatPager(&azblob.ListBlobsFlatOptions{Prefix: &p.SubPath})
			for pager.More() {
				page, err := pager.NextPage(ctx)
				if err != nil {
					return fmt.Errorf("list error: %w", err)
				}
				for _, blob := range page.Segment.BlobItems {
					printBlob(p, *blob.Name, false)
				}
			}
		} else {
			pager := containerClient.NewListBlobsHierarchyPager("/", &container.ListBlobsHierarchyOptions{Prefix: &p.SubPath})
			for pager.More() {
				page, err := pager.NextPage(ctx)
				if err != nil {
					return fmt.Errorf("list error: %w", err)
				}
				for _, prefix := range page.Segment.BlobPrefixes {
					printBlob(p, *prefix.Name, true)
				}
				for _, blob := range page.Segment.BlobItems {
					printBlob(p, *blob.Name, false)
				}
			}
		}

		return nil
	},
}

func init() {
	lsCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively list all blobs")
	lsCmd.Flags().BoolVar(&fullPath, "full-path", false, "Show full blob path (az:// or https)")
	lsCmd.Flags().BoolVar(&pretty, "pretty", false, "Show pretty icons for files and directories")
}

func printBlob(p *azpath.BlobPath, name string, isDir bool) {
	var output string

	if fullPath {
		output = p.BuildFull(name)
	} else {
		output = name
	}

	if pretty {
		if isDir {
			fmt.Printf(" 📁 %s\n", output)
		} else {
			fmt.Printf(" 📄 %s\n", output)
		}
	} else {
		fmt.Println(output)
	}
}
