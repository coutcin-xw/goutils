package nettools

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// 链式方法集合

func (r *Req) SetMethod(method string) *Req {
	r.Method = strings.ToUpper(method)
	return r
}

func (r *Req) Get() *Req    { return r.SetMethod(http.MethodGet) }
func (r *Req) Post() *Req   { return r.SetMethod(http.MethodPost) }
func (r *Req) Put() *Req    { return r.SetMethod(http.MethodPut) }
func (r *Req) Delete() *Req { return r.SetMethod(http.MethodDelete) }

func (r *Req) GetRequest() *http.Request {
	return r.Requests
}

func (r *Req) SetUrl(url string) *Req {
	r.Url = url
	return r
}

func (r *Req) SetParams(params map[string]interface{}) *Req {
	r.Params = params
	return r
}

func (r *Req) SetData(data map[string]interface{}) *Req {
	r.Data = data
	return r
}

func (r *Req) SetHeader(key, value string) *Req {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	r.Headers[key] = value
	return r
}

func (r *Req) SetHeaders(headers map[string]string) *Req {
	r.Headers = headers
	return r
}

func (r *Req) AddCookie(cookie *http.Cookie) *Req {
	r.Cookies = append(r.Cookies, cookie)
	return r
}

func (r *Req) SetTimeout(timeout time.Duration) *Req {
	r.Client.Timeout = timeout
	return r
}

func (r *Req) SetVerify(verify bool) *Req {
	r.Verify = verify
	return r
}

func (r *Req) SetCertPaths(paths []string) *Req {
	r.CertPaths = paths
	return r
}

func (r *Req) SetProxy(proxyURL string) *Req {
	r.Proxy = proxyURL
	return r
}

func (r *Req) AddFile(fieldName, fileName string, file io.Reader, contentType ...string) *Req {
	ct := "application/octet-stream"
	if len(contentType) > 0 {
		ct = contentType[0]
	}

	r.Files = append(r.Files, &RequestFile{
		FieldName:   fieldName,
		FileName:    fileName,
		File:        file,
		ContentType: ct,
	})
	return r
}

// Do 执行HTTP请求
func (r *Req) Do() (*http.Response, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}

	// 构建请求URL
	reqUrl, err := r.buildURL()
	if err != nil {
		return nil, err
	}

	// 构建请求体
	body, contentType, err := r.buildBody()
	if err != nil {
		return nil, err
	}

	// 创建请求对象
	req, err := http.NewRequest(r.Method, reqUrl, body)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	r.setHeaders(req, contentType)

	// 配置HTTP客户端
	if err := r.configureClient(); err != nil {
		return nil, err
	}
	r.Requests = req
	// 执行请求
	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求执行失败: %w", err)
	}

	// 保存响应cookie
	r.saveCookies(resp, req.URL)

	return resp, nil
}

// 私有方法 ---------------------------------------------------

func (r *Req) validate() error {
	if r.Url == "" {
		return fmt.Errorf("请求URL不能为空")
	}
	if r.Method == "" {
		return fmt.Errorf("HTTP方法未设置")
	}
	return nil
}

func (r *Req) buildURL() (string, error) {
	u, err := url.Parse(r.Url)
	if err != nil {
		return "", fmt.Errorf("解析URL失败: %w", err)
	}

	query := u.Query()
	for k, v := range r.Params {
		query.Add(k, fmt.Sprintf("%v", v))
	}
	u.RawQuery = query.Encode()

	return u.String(), nil
}

func (r *Req) buildBody() (io.Reader, string, error) {
	// 处理文件上传
	if len(r.Files) > 0 {
		return r.buildMultipartBody()
	}

	// 处理普通数据
	return r.buildNormalBody()
}

func (r *Req) buildMultipartBody() (io.Reader, string, error) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	// 添加文件字段
	for _, file := range r.Files {
		part, err := writer.CreateFormFile(file.FieldName, filepath.Base(file.FileName))
		if err != nil {
			return nil, "", err
		}
		if _, err := io.Copy(part, file.File); err != nil {
			return nil, "", err
		}
	}

	// 添加数据字段
	for k, v := range r.Data {
		writer.WriteField(k, fmt.Sprintf("%v", v))
	}

	contentType := writer.FormDataContentType()
	writer.Close()
	return buf, contentType, nil
}

func (r *Req) buildNormalBody() (io.Reader, string, error) {
	if r.Data == nil {
		return nil, "", nil
	}

	// 优先使用用户指定的Content-Type
	if contentType, ok := r.Headers["Content-Type"]; ok {
		switch contentType {
		case "application/json":
			jsonData, err := json.Marshal(r.Data)
			return bytes.NewBuffer(jsonData), contentType, err
		case "application/x-www-form-urlencoded":
			formData := url.Values{}
			for k, v := range r.Data {
				formData.Add(k, fmt.Sprintf("%v", v))
			}
			return strings.NewReader(formData.Encode()), contentType, nil
		default:
			return nil, "", fmt.Errorf("不支持的Content-Type: %s", contentType)
		}
	}

	// 默认使用JSON格式
	jsonData, err := json.Marshal(r.Data)
	return bytes.NewBuffer(jsonData), "application/json", err
}

func (r *Req) setHeaders(req *http.Request, contentType string) {
	// 自动设置Content-Type
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// 用户自定义头信息
	for k, v := range r.Headers {
		req.Header.Set(k, v)
	}

	// Cookies
	for _, cookie := range r.Cookies {
		req.AddCookie(cookie)
	}
}

func (r *Req) configureClient() error {
	// 确保Transport存在
	if r.Client.Transport == nil {
		r.Client.Transport = &http.Transport{}
	}

	// 类型断言获取Transport
	transport, ok := r.Client.Transport.(*http.Transport)
	if !ok {
		return fmt.Errorf("不支持的Transport类型")
	}

	// 配置TLS
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}
	transport.TLSClientConfig.InsecureSkipVerify = !r.Verify

	// 在证书配置部分修改为：
	if len(r.CertPaths) > 0 {
		// 使用系统证书池作为基础
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}

		// 添加自定义证书
		for _, path := range r.CertPaths {
			cert, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("读取证书失败: %w", err)
			}
			if !pool.AppendCertsFromPEM(cert) {
				return fmt.Errorf("添加证书到池失败: %s", path)
			}
		}
		transport.TLSClientConfig.RootCAs = pool
	} else {
		// 当不添加证书时保持系统默认
		transport.TLSClientConfig.RootCAs = nil
	}

	// 配置代理
	if r.Proxy != "" {
		proxyURL, err := url.Parse(r.Proxy)
		if err != nil {
			return fmt.Errorf("解析代理URL失败: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return nil
}

func (r *Req) saveCookies(resp *http.Response, url *url.URL) {
	if jar, ok := r.Client.Jar.(*cookiejar.Jar); ok {
		jar.SetCookies(url, resp.Cookies())
	}
}

// 响应处理方法 ---------------------------------------------------

func (r *Req) DoAndGetBody() ([]byte, error) {
	resp, err := r.Do()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP错误状态码: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (r *Req) DoAndUnmarshal(v interface{}) error {
	resp, err := r.Do()
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP错误状态码: %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(v)
}
