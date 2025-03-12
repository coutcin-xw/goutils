package nettools

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

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
	// 修复1: 包含完整的URL路径和参数
	urlPart := req.URL.Path
	if req.URL.RawQuery != "" {
		urlPart += "?" + req.URL.RawQuery
	}
	// 打印请求方法和 URL
	requestDetails.WriteString(fmt.Sprintf("%s %s %s\r\n", req.Method, urlPart, req.Proto))

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

type InterfaceIpInfo struct {
	IfaceName   string
	IfaceIsUp   bool
	IfaceIpNets []net.IPNet
}

func GetIpv6Global() ([]InterfaceIpInfo, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("error fetching interfaces: %v", err)
	}
	var ifaceIpInfo []InterfaceIpInfo
	for _, iface := range interfaces {
		// 检查接口是否启用
		if iface.Flags&net.FlagUp == 0 {
			ifaceIpInfo = append(ifaceIpInfo, InterfaceIpInfo{
				IfaceName:   iface.Name,
				IfaceIsUp:   false,
				IfaceIpNets: nil,
			})
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		var ifaceIpNets []net.IPNet
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP
			if ip.IsLoopback() {
				continue
			}
			if ip.To4() == nil && ip.To16() != nil && ip.IsGlobalUnicast() {
				ifaceIpNets = append(ifaceIpNets, *ipNet)
			}
		}

		ifaceIpInfo = append(ifaceIpInfo, InterfaceIpInfo{
			IfaceName:   iface.Name,
			IfaceIsUp:   true,
			IfaceIpNets: ifaceIpNets,
		})
	}
	return ifaceIpInfo, nil
}
func GetIpv6() ([]InterfaceIpInfo, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("error fetching interfaces: %v", err)
	}
	var ifaceIpInfo []InterfaceIpInfo
	for _, iface := range interfaces {
		// 检查接口是否启用
		if iface.Flags&net.FlagUp == 0 {
			ifaceIpInfo = append(ifaceIpInfo, InterfaceIpInfo{
				IfaceName:   iface.Name,
				IfaceIsUp:   false,
				IfaceIpNets: nil,
			})
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		var ifaceIpNets []net.IPNet
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP
			if ip.To4() == nil && ip.To16() != nil {
				ifaceIpNets = append(ifaceIpNets, *ipNet)
			}
		}

		ifaceIpInfo = append(ifaceIpInfo, InterfaceIpInfo{
			IfaceName:   iface.Name,
			IfaceIsUp:   true,
			IfaceIpNets: ifaceIpNets,
		})
	}
	return ifaceIpInfo, nil
}
func GetIpv4() ([]InterfaceIpInfo, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("error fetching interfaces: %v", err)
	}
	var ifaceIpInfo []InterfaceIpInfo
	for _, iface := range interfaces {
		// 检查接口是否启用
		if iface.Flags&net.FlagUp == 0 {
			ifaceIpInfo = append(ifaceIpInfo, InterfaceIpInfo{
				IfaceName:   iface.Name,
				IfaceIsUp:   false,
				IfaceIpNets: nil,
			})
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		var ifaceIpNets []net.IPNet
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP
			if ip.To4() != nil {
				ifaceIpNets = append(ifaceIpNets, *ipNet)
			}
		}

		ifaceIpInfo = append(ifaceIpInfo, InterfaceIpInfo{
			IfaceName:   iface.Name,
			IfaceIsUp:   true,
			IfaceIpNets: ifaceIpNets,
		})
	}
	return ifaceIpInfo, nil
}

func GetInterFaceInfo(ifaceName string) (*InterfaceIpInfo, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("error fetching interfaces: %v", err)
	}
	var ifaceIpInfo *InterfaceIpInfo
	for _, iface := range interfaces {
		if iface.Name != ifaceName {
			continue
		}
		// 检查接口是否启用
		if iface.Flags&net.FlagUp == 0 {
			ifaceIpInfo = &InterfaceIpInfo{
				IfaceName:   iface.Name,
				IfaceIsUp:   false,
				IfaceIpNets: nil,
			}
			return ifaceIpInfo, nil
		}

		addrs, err := iface.Addrs()
		if err != nil {

			return nil, err
		}
		var ifaceIpNets []net.IPNet
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP
			if ip.To4() != nil || ip.To16() != nil {
				ifaceIpNets = append(ifaceIpNets, *ipNet)
			}
		}
		ifaceIpInfo = &InterfaceIpInfo{
			IfaceName:   iface.Name,
			IfaceIsUp:   true,
			IfaceIpNets: ifaceIpNets,
		}
		return ifaceIpInfo, nil

	}
	return nil, fmt.Errorf("get inface info null")
}

func GetIfaceIpv4(ifaceName string) (*InterfaceIpInfo, error) {
	info, err := GetInterFaceInfo(ifaceName)
	if err != nil {
		return nil, err
	}
	if !info.IfaceIsUp || info == nil {
		return info, nil
	}
	var ipv4 []net.IPNet
	for _, x := range info.IfaceIpNets {
		if x.IP.To4() != nil {
			ipv4 = append(ipv4, x)
		}
	}
	info.IfaceIpNets = ipv4
	return info, nil

}
func GetIfaceIpv4Global(ifaceName string) (*InterfaceIpInfo, error) {
	info, err := GetInterFaceInfo(ifaceName)
	if err != nil {
		return nil, err
	}
	if !info.IfaceIsUp || info == nil {
		return info, nil
	}
	var ipv4 []net.IPNet
	for _, x := range info.IfaceIpNets {
		if x.IP.To4() != nil && x.IP.IsGlobalUnicast() {
			ipv4 = append(ipv4, x)
		}
	}
	info.IfaceIpNets = ipv4
	return info, nil

}
func GetIfaceIpv6(ifaceName string) (*InterfaceIpInfo, error) {
	info, err := GetInterFaceInfo(ifaceName)
	if err != nil {
		return nil, err
	}
	if !info.IfaceIsUp || info == nil {
		return info, nil
	}
	var ipv6 []net.IPNet
	for _, x := range info.IfaceIpNets {
		if x.IP.To4() == nil && x.IP.To16() != nil {
			ipv6 = append(ipv6, x)
		}
	}
	info.IfaceIpNets = ipv6
	return info, nil
}
func GetIfaceIpv6Global(ifaceName string) (*InterfaceIpInfo, error) {
	info, err := GetInterFaceInfo(ifaceName)
	if err != nil {
		return nil, err
	}
	if !info.IfaceIsUp || info == nil {
		return info, nil
	}
	var ipv6 []net.IPNet
	for _, x := range info.IfaceIpNets {
		if x.IP.To4() == nil && x.IP.To16() != nil && x.IP.IsGlobalUnicast() {
			ipv6 = append(ipv6, x)
		}
	}
	info.IfaceIpNets = ipv6
	return info, nil
}
