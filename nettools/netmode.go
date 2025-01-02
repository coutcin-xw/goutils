package nettools

import (
	"os"
)

type ReuqestFiles struct {
	FileName    string
	File        *os.File
	ContentType string
}
type RequestOptions struct {
	Method    string
	URL       string
	Params    map[string]interface{}
	Data      map[string]interface{}
	Files     *ReuqestFiles
	Verify    bool
	CertPaths []string // Updated to specify certificate paths
	Proxies   map[string]string
	Headers   map[string]string
}
