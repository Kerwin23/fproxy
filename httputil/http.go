package httputil

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func DoHttpGet(url, proxy string, headers map[string]string, maxBodyLength int) ([]byte, error) {
	return doHttpMethod(url, "GET", proxy, headers, maxBodyLength)
}

func GetForCheck(url, proxy, checkWord string, headers map[string]string, maxBodyLength int) bool {
	bs, err := DoHttpGet(url, proxy, headers, maxBodyLength)
	if err != nil {
		glog.Errorln("get for check error: ", url, " ", proxy, " ", err)
		return false
	}
	html := string(bs)
	b := strings.Contains(html, checkWord)
	glog.Infoln("get for check: ", url, " ", proxy, " ", checkWord, " ", b)
	return b
}

func DoHttpHead(url, proxy string, headers map[string]string) (*http.Response, error) {
	return doHttpRequest(url, "HEAD", proxy, headers)
}

func HeadForCheck(url, proxy string, headers map[string]string, statusCode int) bool {
	res, err := DoHttpHead(url, proxy, headers)
	if err != nil {
		return false
	}
	return res.StatusCode == statusCode
}

func doHttpMethod(url, method, proxy string, headers map[string]string, maxBodyLength int) ([]byte, error) {
	res, err := doHttpRequest(url, method, proxy, headers)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return readFromHttpResponse(res, maxBodyLength)
}

func doHttpRequest(url, method, proxy string, headers map[string]string) (*http.Response, error) {
	client := createHttpClient(proxy)
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	setHttpHeaders(req, headers)
	return client.Do(req)
}

func readFromHttpResponse(response *http.Response, maxBodyLength int) ([]byte, error) {
	reader := bufio.NewReader(response.Body)
	readBuf := make([]byte, 4096)
	writeBuf := make([]byte, 4096)
	bytesBuffer := bytes.NewBuffer(writeBuf)
	totallen := 0
	for {
		buflen, err := reader.Read(readBuf)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}
		totallen = totallen + buflen
		if maxBodyLength > 0 && totallen > maxBodyLength {
			return nil, errors.New("http body length exceed max length[" + fmt.Sprintf("%d", maxBodyLength) + "]")
		}
		bytesBuffer.Write(readBuf[0:buflen])
	}
	return bytesBuffer.Bytes(), nil
}

func createHttpClient(proxy string) *http.Client {
	if proxy == "" {
		return &http.Client{}
	}
	urli := url.URL{}
	proxyUrl := "http://" + proxy
	urlproxy, _ := urli.Parse(proxyUrl)
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(urlproxy)}, Timeout: time.Duration(10 * time.Second)}
	return client
}

func setHttpHeaders(request *http.Request, headers map[string]string) {
	if headers == nil {
		return
	}
	for name, value := range headers {
		request.Header.Add(name, value)
	}
}
