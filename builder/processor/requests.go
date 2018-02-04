package processor

import (
	"encoding/xml"
	"errors"
	"io/ioutil"
)

type CheckRequest struct {
	Type      string
	Url       string
	Word      string
	UserAgent string
	MaxLength int
}

type CheckRequestsXml struct {
	XMLName  xml.Name          `xml:"requests"`
	Requests []CheckRequestXml `xml:"request"`
}

type CheckRequestXml struct {
	Type      string `xml:"type,attr"`
	Url       string `xml:"url,attr"`
	Word      string `xml:"word,attr"`
	UA        string `xml:"ua,attr"`
	MaxLength int    `xml:"max-length,attr"`
}

func ParseRequestXml(path string) ([]CheckRequest, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	requestsXml := CheckRequestsXml{}
	err = xml.Unmarshal(content, &requestsXml)
	if err != nil {
		return nil, err
	}
	return xmlToObjects(requestsXml)
}

func xmlToObjects(requestsXml CheckRequestsXml) ([]CheckRequest, error) {
	requestXmls := requestsXml.Requests
	if requestXmls == nil || len(requestXmls) == 0 {
		return nil, errors.New("check request is empty")
	}
	reqLen := len(requestXmls)
	requests := make([]CheckRequest, reqLen)
	for i, requestXml := range requestXmls {
		request := CheckRequest{Type: requestXml.Type, Url: requestXml.Url, Word: requestXml.Word, UserAgent: requestXml.UA, MaxLength: requestXml.MaxLength}
		requests[i] = request
	}
	return requests, nil
}
