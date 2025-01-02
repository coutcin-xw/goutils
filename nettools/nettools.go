package nettools

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

func NewRequestOptions() *RequestOptions {
	return &RequestOptions{
		Method:    "GET",
		Verify:    false,
		Params:    make(map[string]interface{}),
		Data:      make(map[string]interface{}),
		CertPaths: []string{},
		Proxies:   make(map[string]string),
		Headers:   make(map[string]string),
	}
}

func Request(opts *RequestOptions) (*http.Response, error) {
	// Process URL and query parameters
	urlStr := opts.URL
	if len(opts.Params) > 0 {
		query := url.Values{}
		for key, value := range opts.Params {
			query.Add(key, value.(string))
		}
		if strings.Contains(urlStr, "?") {
			urlStr += "&" + query.Encode()
		} else {
			urlStr += "?" + query.Encode()
		}
	}

	// Create HTTP client with optional proxy and TLS configurations
	transport := &http.Transport{}

	if len(opts.CertPaths) == 2 {
		// Load custom certificates
		cert, err := tls.LoadX509KeyPair(opts.CertPaths[0], opts.CertPaths[1])
		if err != nil {
			return nil, err
		}
		transport.TLSClientConfig = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: !opts.Verify,
		}
	} else if !opts.Verify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	// Configure proxy settings
	if opts.Proxies != nil {
		if proxyURL, ok := opts.Proxies["http"]; ok {
			proxy, err := url.Parse(proxyURL)
			if err != nil {
				return nil, err
			}
			transport.Proxy = http.ProxyURL(proxy)
		} else if proxyURL, ok := opts.Proxies["https"]; ok {
			proxy, err := url.Parse(proxyURL)
			if err != nil {
				return nil, err
			}
			transport.Proxy = http.ProxyURL(proxy)
		} else if proxyURL, ok := opts.Proxies["socks5"]; ok {
			proxy, err := url.Parse(proxyURL)
			if err != nil {
				return nil, err
			}
			transport.Proxy = http.ProxyURL(proxy)
		}
	}

	client := &http.Client{Transport: transport}

	// Prepare request body
	var body io.Reader
	contentType := "application/json"

	if opts.Files != nil && opts.Files.File != nil {
		// Multipart form for file upload
		buffer := &bytes.Buffer{}
		writer := multipart.NewWriter(buffer)

		// Add file part
		fileWriter, err := writer.CreateFormFile("file", opts.Files.FileName)
		if err != nil {
			return nil, err
		}
		if _, err = io.Copy(fileWriter, opts.Files.File); err != nil {
			return nil, err
		}

		// Add additional data
		for key, value := range opts.Data {
			_ = writer.WriteField(key, value.(string))
		}

		writer.Close()
		body = buffer
		contentType = writer.FormDataContentType()
	} else if opts.Data != nil {
		// JSON payload
		jsonData, err := json.Marshal(opts.Data)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(jsonData)
	}

	// Create HTTP request
	req, err := http.NewRequest(strings.ToUpper(opts.Method), urlStr, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)

	// Add custom headers
	for key, value := range opts.Headers {
		req.Header.Set(key, value)
	}
	// reqs, _ := ReadRequest(req, true)
	// fmt.Print(string(reqs))
	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func GetURIExtension(URI string) (string, error) {
	tmpURI, err := RemoveQueryParams(URI)
	if err != nil {
		return "", err
	}
	// 获取文件的扩展名
	ext := filepath.Ext(tmpURI)
	return ext, nil
}

func RemoveQueryParams(urlStr string) (string, error) {
	parsedUrl, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	// 清空查询参数
	parsedUrl.RawQuery = ""

	// 清空哈希部分
	parsedUrl.Fragment = ""

	return parsedUrl.String(), nil
}

// ReadRequest 打印请求的所有内容并返回结果
func ReadRequest(req *http.Request, isCut bool) ([]byte, error) {
	// 使用一个缓冲区保存请求内容
	var requestDetails bytes.Buffer
	var bodyCopy bytes.Buffer
	var body bytes.Buffer

	// 打印请求方法和 URL
	requestDetails.WriteString(fmt.Sprintf("%s %s %s\r\n", req.Method, req.URL.Path, req.Proto))

	req.Header.Add("Host", req.URL.Host)
	// 打印请求头
	for key, values := range req.Header {
		for _, value := range values {
			requestDetails.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
		}
	}

	// 如果请求体不为空，才进行读取
	if req.Body != nil {
		// 创建一个 TeeReader 复制数据
		tee := io.TeeReader(req.Body, &bodyCopy)

		// 根据需要逐块读取请求体（避免内存占用过高）
		chunkSize := 8192
		buf := make([]byte, chunkSize)
		var totalRead int
		var isRead int = 1
		for {
			n, err := tee.Read(buf)
			if n > 0 {
				totalRead += n
				if isCut && totalRead > 100 {
					if isRead >= 1 {
						body.Write(buf[:100-totalRead+n]) // 只截断到 100 字节
						isRead--
					} else {
						continue
					}

				} else {
					body.Write(buf[:n])
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("failed to read request body: %v", err)
			}
		}
		// 重新设置请求体，确保后续逻辑能继续使用
		req.Body = io.NopCloser(&bodyCopy)
		// 将读取的请求体内容放到缓冲区
		if body.Len() > 0 {
			requestDetails.WriteString(fmt.Sprintf("\r\n%s", body.String()))
		} else {
			requestDetails.WriteString("\r\n")
		}
	} else {
		// 如果没有请求体，直接写入换行符
		requestDetails.WriteString("\r\n")
	}

	// 返回请求内容的字节数组
	return requestDetails.Bytes(), nil
}

// 打印响应内容
func ReadResponse(resp *http.Response, isCut bool) ([]byte, error) {
	// 使用一个缓冲区保存响应内容
	var responseDetails bytes.Buffer
	var bodyCopy bytes.Buffer
	var body bytes.Buffer
	// 打印响应状态码和状态文本
	responseDetails.WriteString(fmt.Sprintf("%s %d %s\r\n", resp.Proto, resp.StatusCode, http.StatusText(resp.StatusCode)))

	// 打印响应头
	for key, values := range resp.Header {
		for _, value := range values {
			responseDetails.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
		}
	}
	// 如果请求体不为空，才进行读取
	if resp.Body != nil {
		// 创建一个 TeeReader 复制数据
		tee := io.TeeReader(resp.Body, &bodyCopy)

		// 根据需要逐块读取（避免内存占用过高）
		chunkSize := 8192
		buf := make([]byte, chunkSize)
		var totalRead int
		var isRead int = 1
		for {
			n, err := tee.Read(buf)
			if n > 0 {
				totalRead += n
				if isCut && totalRead > 100 {
					if isRead >= 1 {
						body.Write(buf[:100-totalRead+n]) // 只截断到 100 字节
						isRead--
					} else {
						continue
					}
				} else {
					body.Write(buf[:n])
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {

				return nil, fmt.Errorf("failed to read response body: %v", err)
			}

		}
		// 重新设置响应体，确保后续逻辑能继续使用
		resp.Body = io.NopCloser(&bodyCopy)
		// 将读取的响应体内容放到缓冲区
		if body.Len() > 0 {
			responseDetails.WriteString(fmt.Sprintf("\r\n%s", body.String()))
		} else {
			responseDetails.WriteString("\r\n")
		}
	} else {
		// 如果没有请求体，直接写入换行符
		responseDetails.WriteString("\r\n")
	}

	return responseDetails.Bytes(), nil
}

// ReadResponseBody 读取 HTTP 响应体并返回为字符串
func ReadResponseBody(resp *http.Response) ([]byte, error) {
	// 使用一个缓冲区保存响应内容
	var responseDetails bytes.Buffer

	// 创建一个临时缓冲区保存响应体内容
	var bodyBuffer bytes.Buffer

	// 使用 io.TeeReader 同时读取响应体并复制内容到 bodyBuffer
	reader := io.TeeReader(resp.Body, &bodyBuffer)
	// 重新设置响应体，确保后续逻辑能继续使用
	resp.Body = io.NopCloser(&bodyBuffer)
	// 分块读取响应体内容，避免一次性读取过大的文件
	chunkSize := 8192 // 每次读取 8KB
	buf := make([]byte, chunkSize)

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			responseDetails.Write(buf[:n]) // 将内容写入字符串缓冲区
		}
		if err == io.EOF {
			break // 读取完成
		}
		if err != nil {

			return nil, fmt.Errorf("failed to read response body: %v", err)
		}
	}

	// 返回读取到的响应体
	return responseDetails.Bytes(), nil
}

// 判断 IP 是否在列表中
func IsIPInList(ip string, ipList []string) bool {

	for _, cidr := range ipList {
		if strings.Contains(cidr, "/") {
			// CIDR 格式处理
			if IsIPInCIDR(ip, cidr) {
				return true
			}
		} else if ip == cidr {
			// 精确匹配 IP
			return true
		}
	}
	return false
}

// 检查 IP 是否在 CIDR 范围内
func IsIPInCIDR(ip string, cidr string) bool {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false // CIDR 格式错误
	}
	parsedIP := net.ParseIP(ip)
	return ipNet.Contains(parsedIP)
}
