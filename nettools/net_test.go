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

	// 创建请求
	// req := NewRequest().
	// 	SetUrl("https://www.baidu.com/data").
	// 	Get().
	// 	SetParams(map[string]interface{}{
	// 		"page": 1,
	// 		"size": 20,
	// 	}).
	// 	SetHeader("Authorization", "Bearer token")

	// resp, _ := req.Do()
	// reqs, _ := ReadRequest(req.Requests, false)
	// 上传文件
	file, _ := os.Open("test.txt")
	resp, err := NewRequest().
		SetUrl("https://api.example.com/upload").
		Post().
		AddFile("file", "test.txt", file).
		SetData(map[string]interface{}{
			"description": "example file",
		}).
		Do()
	if err != nil {

	}
	defer resp.Body.Close()

	// body, _ := io.ReadAll(resp.Body)
	body, _ := ReadResponse(resp, true)
	// body2, _ := ReadResponse(resp, false)
	println(string(body))

	// println(string(reqs))
}
