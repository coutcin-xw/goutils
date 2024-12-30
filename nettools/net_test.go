package nettools

import (
	"testing"

	"github.com/coutcin-xw/go-logs"
)

func TestNetTools_Console(t *testing.T) {
	url := "http://baidu.com/sad/asd/a.jsp?asd=1#asdd"
	url1, _ := RemoveQueryParams(url)
	ext, _ := GetURIExtension(url1)
	logs.Log.Console(ext)
}
