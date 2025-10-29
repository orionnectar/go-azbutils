package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/orionnectar/go-azbutils/internal/azure"
	"github.com/orionnectar/go-azbutils/internal/config"
	"github.com/spf13/cobra"
)

var (
	dryRun bool
)

var cpCmd = &cobra.Command{
	Use:   "cp <source> <destination>",
	Short: "Copy files or directories to Azure Blob Storage",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		src := args[0]
		dst := args[1]

		if strings.HasPrefix(dst, "az://") {
			info, err := os.Stat(src)
			if err != nil {
				return fmt.Errorf("failed to access source: %w", err)
			}

			if info.IsDir() {
				if !recursive {
					return fmt.Errorf("'%s' is a directory. Use -r or --recursive to upload recursively", src)
				}
				return uploadDirectory(src, dst)
			}

			return uploadFile(src, dst)
		}

		return fmt.Errorf("destination must be an Azure path (az://account//container/blob)")
	},
}

func uploadFile(localPath, dst string) error {
	path := strings.TrimPrefix(dst, "az://")
	parts := strings.SplitN(path, "//", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid destination format, expected az://account//container/blob")
	}
	accountName := parts[0]
	containerAndBlob := parts[1]

	containerParts := strings.SplitN(containerAndBlob, "/", 2)
	if len(containerParts) != 2 {
		return fmt.Errorf("destination must include both container and blob name")
	}
	containerName := containerParts[0]
	blobName := containerParts[1]

	if dryRun {
		fmt.Printf("[dry-run] Would upload %s → az://%s//%s/%s\n", localPath, accountName, containerName, blobName)
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	acctCfg := cfg.Accounts[accountName]
	if acctCfg == nil {
		return fmt.Errorf("no account found in config for '%s'", accountName)
	}

	client, err := azure.NewClientFromConfigAccount(acctCfg)
	if err != nil {
		return fmt.Errorf("failed to create Azure client: %w", err)
	}

	containerClient := client.ServiceClient().NewContainerClient(containerName)
	blobClient := containerClient.NewBlockBlobClient(blobName)

	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Printf("Uploading %s → az://%s//%s/%s\n", localPath, accountName, containerName, blobName)
	_, err = blobClient.UploadStream(ctx, file, &azblob.UploadStreamOptions{})
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	fmt.Println("Upload complete")
	return nil
}

func uploadDirectory(localDir, dst string) error {
	fmt.Printf("Uploading directory %s recursively...\n", localDir)

	err := filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Compute relative path
		rel, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}

		// Append file name to destination path
		dstPath := strings.TrimSuffix(dst, "/") + "/" + rel
		dstPath = filepath.ToSlash(dstPath)

		return uploadFile(path, dstPath)
	})

	if err != nil {
		return fmt.Errorf("directory upload failed: %w", err)
	}

	if dryRun {
		fmt.Println("[dry-run] Directory upload simulated — no files uploaded.")
	} else {
		fmt.Println("Directory upload complete")
	}
	return nil
}

func init() {
	cpCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Upload directories recursively")
	cpCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview upload actions without performing them")
}
