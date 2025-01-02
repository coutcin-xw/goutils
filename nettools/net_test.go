package nettools

import (
	"os"
	"testing"
)

func TestNetTools_Console(t *testing.T) {
	// url := "http://baidu.com/sad/asd/a.jsp?asd=1#asdd"
	// url1, _ := RemoveQueryParams(url)
	// ext, _ := GetURIExtension(url1)
	// logs.Log.Console(ext)

	opts := NewRequestOptions()
	opts.Method = "POST"
	opts.URL = "https://baidu.com/api"
	opts.Params["query"] = "test"
	opts.Data["key"] = "value"
	opts.Headers["Authorization"] = "Bearer token"
	// opts.CertPaths = []string{"/path/to/cert.crt", "/path/to/key.key"}
	opts.Proxies = map[string]string{
		"http": "http://127.0.0.1:7897",
	}

	file, _ := os.Open("/home/coutcin/Downloads/未命名绘图.png")
	defer file.Close()
	opts.Files = &ReuqestFiles{
		FileName:    "example.png",
		File:        file,
		ContentType: "text/plain",
	}

	resp, err := Request(opts)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// body, _ := io.ReadAll(resp.Body)
	body, _ := ReadResponse(resp, true)
	body2, _ := ReadResponse(resp, false)
	println(string(body))

	println(string(body2))
}
