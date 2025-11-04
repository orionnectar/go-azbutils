package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/orionnectar/go-azbutils/internal/azpath"
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
			return fmt.Errorf("no account found in config for '%s'", p.Account)
		}

		client, err := azure.NewClientFromConfigAccount(acctCfg)
		if err != nil {
			return fmt.Errorf("failed to create Azure client: %w", err)
		}

		containerClient := client.ServiceClient().NewContainerClient(p.Container)
		blobClient := containerClient.NewBlockBlobClient(p.SubPath)

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

		fmt.Printf("Downloading blob '%s' → %s\n", p.SubPath, outputFile)
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
