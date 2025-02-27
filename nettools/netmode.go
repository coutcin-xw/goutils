package nettools

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"
)

// Req 表示一个可链式调用的HTTP请求构建器
type Req struct {
	Client    *http.Client
	Requests  *http.Request
	Method    string
	Url       string
	Params    map[string]interface{}
	Data      map[string]interface{}
	Headers   map[string]string
	Cookies   []*http.Cookie
	Files     []*RequestFile // 改为支持多个文件
	Verify    bool
	CertPaths []string
	Proxy     string // 改为单个代理URL
	Timeout   time.Duration
}

// RequestFile 表示要上传的文件
type RequestFile struct {
	FieldName   string
	FileName    string
	File        io.Reader
	ContentType string
}

// NewRequest 创建新的请求对象
func NewRequest() *Req {
	jar, _ := cookiejar.New(nil)
	return &Req{
		Client: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
	}
}
