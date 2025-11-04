package azpath

import (
	"fmt"
	"regexp"
	"strings"
)

type BlobPath struct {
	Account   string
	Container string
	SubPath   string
	Type      string // "az" or "url"
}

// Parse takes an Azure Blob URL or az:// path and normalizes it into BlobPath
func Parse(input string) (*BlobPath, error) {
	if strings.HasPrefix(input, "https://") {
		re := regexp.MustCompile(`https://([^.]+)\.blob\.core\.windows\.net/([^/]+)(/(.*))?`)
		matches := re.FindStringSubmatch(input)
		if len(matches) >= 3 {
			subpath := ""
			if len(matches) > 4 && matches[4] != "" {
				subpath = matches[4]
			}
			return &BlobPath{
				Account:   matches[1],
				Container: matches[2],
				SubPath:   subpath,
				Type:      "url",
			}, nil
		}
		return nil, fmt.Errorf("invalid Azure blob URL: %s", input)
	}

	if strings.HasPrefix(input, "az://") {
		path := strings.TrimPrefix(input, "az://")
		parts := strings.SplitN(path, "//", 2)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid az path format. expected az://<account>//<container>")
		}
		containerParts := strings.SplitN(parts[1], "/", 2)
		subpath := ""
		if len(containerParts) == 2 {
			subpath = containerParts[1]
		}
		return &BlobPath{
			Account:   parts[0],
			Container: containerParts[0],
			SubPath:   subpath,
			Type:      "az",
		}, nil
	}

	return nil, fmt.Errorf("unsupported path format: %s", input)
}

// BuildFull formats a blob name back into a full path depending on input type
func (p *BlobPath) BuildFull(blobName string) string {
	switch p.Type {
	case "url":
		return fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", p.Account, p.Container, blobName)
	case "az":
		return fmt.Sprintf("az://%s//%s/%s", p.Account, p.Container, blobName)
	default:
		return blobName
	}
}
