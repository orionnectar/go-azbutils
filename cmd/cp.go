package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/orionnectar/go-azbutils/internal/azpath"
	"github.com/orionnectar/go-azbutils/internal/azure"
	"github.com/orionnectar/go-azbutils/internal/config"
	"github.com/spf13/cobra"
)

var (
	dryRun bool
)

var cpCmd = &cobra.Command{
	Use:   "cp <source> <destination>",
	Short: "Copy a local file or directory to Azure Blob Storage",
	Long: `Upload a single file or an entire directory to Azure Blob Storage.

Examples:
  # Upload a single file
  azbutils cp ./myfile.txt az://myaccount//mycontainer/myfile.txt

  # Upload a directory recursively
  azbutils cp ./myfolder az://myaccount//mycontainer/myfolder -r

  # Dry run (show what would be uploaded)
  azbutils cp ./data az://myaccount//container/data -r --dry-run
`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		src := args[0]
		dst := args[1]

		info, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf("failed to access source: %w", err)
		}

		if info.IsDir() && !recursive {
			return fmt.Errorf("'%s' is a directory. Use -r or --recursive to upload recursively", src)
		}

		// Parse the destination path (az:// or Azure URL)
		p, err := azpath.Parse(dst)
		if err != nil {
			return fmt.Errorf("invalid destination path: %w", err)
		}

		if info.IsDir() {
			return uploadDirectory(src, p)
		}
		return uploadFile(src, p)
	},
}

func uploadFile(localPath string, p *azpath.BlobPath) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	acctCfg := cfg.Accounts[p.Account]
	if acctCfg == nil {
		return fmt.Errorf("no account found in config for '%s'", p.Account)
	}

	client, err := azure.NewClientFromConfigAccount(acctCfg)
	if err != nil {
		return fmt.Errorf("failed to create Azure client: %w", err)
	}

	containerClient := client.ServiceClient().NewContainerClient(p.Container)
	blobClient := containerClient.NewBlockBlobClient(p.SubPath)

	if dryRun {
		fmt.Printf("[dry-run] Would upload %s → %s\n", localPath, p.BuildFull(p.SubPath))
		return nil
	}

	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Printf("Uploading %s → %s\n", localPath, p.BuildFull(p.SubPath))
	_, err = blobClient.UploadStream(ctx, file, &azblob.UploadStreamOptions{})
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	return nil
}

func uploadDirectory(localDir string, p *azpath.BlobPath) error {
	fmt.Printf("Uploading directory %s recursively...\n", localDir)

	err := filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}

		dstPath := p.SubPath
		if dstPath != "" {
			dstPath += "/"
		}
		dstPath += filepath.ToSlash(rel)

		dstBlob := &azpath.BlobPath{
			Account:   p.Account,
			Container: p.Container,
			SubPath:   dstPath,
			Type:      p.Type,
		}

		return uploadFile(path, dstBlob)
	})
	if err != nil {
		return fmt.Errorf("directory upload failed: %w", err)
	}

	if dryRun {
		fmt.Println("[dry-run] Directory upload simulated — no files uploaded.")
	} else {
		fmt.Println("Directory upload complete.")
	}
	return nil
}

func init() {
	cpCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Upload directories recursively")
	cpCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview upload actions without performing them")
}
