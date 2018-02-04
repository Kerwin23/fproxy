package builder

import (
	"errors"
	"github.com/golang/glog"
	xp "gopkg.in/xmlpath.v2"
	"regexp"
	"strconv"
	"strings"
)

const CRAW_ENTRY_TYPE_CTN_REGEX = "content:regex"
const CRAW_ENTRY_TYPE_CTN_XPATH = "content:xpath"
const CRAW_ENTRY_TYPE_CTN_STATIC = "content:static"
const CRAW_ENTRY_TYPE_PAGE_REGEX = "page:regex"
const CRAW_ENTRY_TYPE_PAGE_XPATH = "page:xpath"
const CRAW_ENTRY_TYPE_PAGE_STATIC = "page:static"
const CRAW_RESULT_TYPE_CONTENT = "content"
const CRAW_RESULT_TYPE_URL = "url"
const CRAW_RESULT_TYPE_EMPTY = "empty"

type CrawEntry struct {
	Name    string
	Type    string
	Regex   string
	Xpath   string
	Value   string
	MaxPage int
}

type CrawTemplate struct {
	Name    string
	Entries []CrawEntry
}

type CrawResult struct {
	Name  string
	Type  string
	Value string
}

func ProcessCrawTemplate(html string, template CrawTemplate) ([]CrawResult, error) {
	entries := template.Entries
	entrySize := len(entries)
	results := make([]CrawResult, entrySize)
	for i, entry := range entries {
		result, err := ProcessCrawEntry(html, entry)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func ProcessCrawEntry(html string, entry CrawEntry) (CrawResult, error) {
	entryType := entry.Type
	switch {
	case entryType == CRAW_ENTRY_TYPE_CTN_REGEX || entryType == CRAW_ENTRY_TYPE_PAGE_REGEX:
		return ProcessRegexEntry(html, entry)
	case entryType == CRAW_ENTRY_TYPE_CTN_XPATH || entryType == CRAW_ENTRY_TYPE_PAGE_XPATH:
		return ProcessXpathEntry(html, entry)
	case entryType == CRAW_ENTRY_TYPE_CTN_STATIC || entryType == CRAW_ENTRY_TYPE_PAGE_STATIC:
		return ProcessStaticEntry(html, entry)
	default:
		return CrawResult{Type: CRAW_RESULT_TYPE_EMPTY}, errors.New("unknown entry type")
	}
}

//process regex entry
func ProcessRegexEntry(html string, entry CrawEntry) (CrawResult, error) {
	if (entry.Type != CRAW_ENTRY_TYPE_CTN_REGEX && entry.Type != CRAW_ENTRY_TYPE_PAGE_REGEX) || entry.Regex == "" {
		return CrawResult{Type: CRAW_RESULT_TYPE_EMPTY}, errors.New("error craw entry")
	}
	entryValue := entry.Value
	locates := fetchLocateExpression(entryValue)
	regex, err := regexp.Compile(entry.Regex)
	if err != nil {
		glog.Errorln("compile regex[", entry.Regex, "] error: ", err)
		return CrawResult{Type: CRAW_RESULT_TYPE_EMPTY}, errors.New("error entry regex")
	}
	result := regex.FindAllStringSubmatch(html, -1)
	resultStr := ""
	for _, arr := range result {
		arrLen := len(arr)
		valueStr := entryValue
		for _, locate := range locates {
			if locate >= arrLen {
				continue
			}
			oldStr := "#" + strconv.Itoa(locate) + "#"
			newStr := arr[locate]
			valueStr = strings.Replace(valueStr, oldStr, newStr, -1)
		}
		resultStr = resultStr + "," + valueStr
	}
	targetType := CRAW_RESULT_TYPE_CONTENT
	if entry.Type == CRAW_ENTRY_TYPE_PAGE_REGEX {
		targetType = CRAW_RESULT_TYPE_URL
	}
	return CrawResult{Name: entry.Name, Type: targetType, Value: strings.TrimPrefix(resultStr, ",")}, nil
}

//process xpath entry
func ProcessXpathEntry(html string, entry CrawEntry) (CrawResult, error) {
	if (entry.Type != CRAW_ENTRY_TYPE_CTN_XPATH && entry.Type != CRAW_ENTRY_TYPE_PAGE_XPATH) || entry.Xpath == "" {
		return CrawResult{Type: CRAW_RESULT_TYPE_EMPTY}, errors.New("error craw entry")
	}
	targetType := CRAW_RESULT_TYPE_CONTENT
	if entry.Type == CRAW_ENTRY_TYPE_PAGE_XPATH {
		targetType = CRAW_RESULT_TYPE_URL
	}
	entryValue := entry.Value
	xpath := xp.MustCompile(entry.Xpath)
	htmlReader := strings.NewReader(html)
	xnode, err := xp.Parse(htmlReader)
	if err != nil {
		return CrawResult{Type: CRAW_RESULT_TYPE_EMPTY}, errors.New("xpath parse html error")
	}
	if value, ok := xpath.String(xnode); ok {
		targetValue := strings.Replace(entryValue, "#$#", value, -1)
		return CrawResult{Name: entry.Name, Type: targetType, Value: targetValue}, nil
	}
	return CrawResult{Type: CRAW_RESULT_TYPE_EMPTY}, errors.New("can't fetch xpath data: " + entry.Xpath)
}

//process static entry
func ProcessStaticEntry(html string, entry CrawEntry) (CrawResult, error) {
	if (entry.Type != CRAW_ENTRY_TYPE_CTN_STATIC && entry.Type != CRAW_ENTRY_TYPE_PAGE_STATIC) || entry.Value == "" {
		return CrawResult{Type: CRAW_RESULT_TYPE_EMPTY}, errors.New("error craw entry")
	}
	targetType := CRAW_RESULT_TYPE_CONTENT
	targetValue := entry.Value
	if entry.Type == CRAW_ENTRY_TYPE_PAGE_STATIC {
		targetType = CRAW_RESULT_TYPE_URL
		if entry.MaxPage <= 0 {
			return CrawResult{Type: CRAW_RESULT_TYPE_EMPTY}, errors.New("static page entry error: error max page")
		}
		pageurls := ""
		for i := 0; i < entry.MaxPage; i++ {
			pageurl := strings.Replace(entry.Value, "#page#", strconv.Itoa(i+1), -1)
			pageurls = pageurls + "," + pageurl
		}
		targetValue = strings.TrimPrefix(pageurls, ",")
	}
	return CrawResult{Name: entry.Name, Type: targetType, Value: targetValue}, nil
}

func fetchLocateExpression(src string) []int {
	if src == "" {
		return []int{0}
	}
	regex, _ := regexp.Compile("#(\\d+)#")
	result := regex.FindAllStringSubmatch(src, -1)
	tmpSInt := ""
	for _, arr := range result {
		vi := arr[1]
		if vi != "" {
			tmpSInt = tmpSInt + "," + vi
		}
	}
	if tmpSInt == "" {
		return []int{0}
	}
	tmpSInt = strings.TrimPrefix(tmpSInt, ",")
	tmpSIntArr := strings.Split(tmpSInt, ",")
	locates := make([]int, len(tmpSIntArr))
	for i, sInt := range tmpSIntArr {
		vInt, err := strconv.Atoi(sInt)
		if err != nil {
			glog.Errorln("error locate string[", sInt, "]: ", err)
			continue
		}
		locates[i] = vInt
	}
	return locates
}
