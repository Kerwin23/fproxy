package httputil

import (
	"fmt"
	"testing"
)

func TestHttpProxy(t *testing.T) {
	bs, err := DoHttpGet("http://www.baidu.com", "127.0.0.1:1234", nil, -1)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(bs))
}
