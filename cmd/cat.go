package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/orionnectar/go-azbutils/internal/azure"
	"github.com/orionnectar/go-azbutils/internal/config"
	"github.com/spf13/cobra"
)

var outputFile string

var catCmd = &cobra.Command{
	Use:   "cat <az://account//container/blob>",
	Short: "Print the contents of a blob or save it to a local file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		src := args[0]
		if !strings.HasPrefix(src, "az://") {
			return fmt.Errorf("source must be an Azure blob path (az://account//container/blob)")
		}

		path := strings.TrimPrefix(src, "az://")
		parts := strings.SplitN(path, "//", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid blob path format. Expected az://account//container/blob")
		}

		accountName := parts[0]
		containerAndBlob := parts[1]

		containerParts := strings.SplitN(containerAndBlob, "/", 2)
		if len(containerParts) != 2 {
			return fmt.Errorf("blob path must include both container and blob name")
		}

		containerName := containerParts[0]
		blobName := containerParts[1]

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

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		resp, err := blobClient.DownloadStream(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to download blob: %w", err)
		}

		reader := resp.NewRetryReader(ctx, nil)
		defer reader.Close()

		if outputFile == "" {
			// Stream to stdout
			_, err = io.Copy(os.Stdout, reader)
			if err != nil {
				return fmt.Errorf("failed to stream blob: %w", err)
			}
			return nil
		}

		// Save to file
		out, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer out.Close()

		fmt.Printf("Downloading blob '%s' → %s\n", blobName, outputFile)
		_, err = io.Copy(out, reader)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		fmt.Println("Blob saved successfully")
		return nil
	},
}

func init() {
	catCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write blob contents to a local file instead of stdout")
}
