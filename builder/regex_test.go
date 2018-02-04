package builder

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"testing"
)

func TestRegexForCraw(t *testing.T) {
	pattern := "<td>(.*?)</td>\\s+<td>(\\d{1,5}?)</td>\\s+<td>高匿代理IP</td>"
	regex, err := regexp.Compile(pattern)
	if err != nil {
		t.Error(err)
	}
	// result := regex.FindAllStringSubmatch("#1#:#2#", -1)
	// fmt.Println(result)
	// t.Log(result)
	bs, err := ioutil.ReadFile("E:\\test\\360.html")
	if err != nil {
		t.Error(err)
	}
	html := string(bs)
	result := regex.FindAllStringSubmatch(html, -1)
	fmt.Println(result)
	t.Log(result)
}
