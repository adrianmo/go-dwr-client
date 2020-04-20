package dwr

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

type DWRClient struct {
	httpClient      *http.Client
	host            string
	schema          string
	batchID         int
	initialized     bool
	scriptSessionID string
	params          map[string]string
}

func NewDWRClient(host, schema string, params map[string]string) (*DWRClient, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Jar: jar,
	}

	dwrClient := &DWRClient{
		httpClient:  httpClient,
		host:        host,
		schema:      schema,
		batchID:     0,
		initialized: false,
		params:      params,
	}

	err = dwrClient.init()
	if err != nil {
		return nil, err
	}

	return dwrClient, nil
}

func (dwr *DWRClient) init() error {
	dwr.batchID = 0
	script := "__System"
	method := "generateId"
	dwr.initialized = true

	res, err := dwr.request("", script, method, nil, nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	bodyString := string(bodyBytes)

	re := regexp.MustCompile(`handleCallback\("\w+",\s*"\w+",\s*"(.+?)"\);`)
	s := re.FindAllStringSubmatch(bodyString, -1)
	if s == nil || len(s[0]) < 2 {
		return fmt.Errorf("Could not find DWRSESSIONID in response body")
	}
	dwrSessionID := s[0][1]
	dwr.setSession(dwrSessionID)
	return nil
}

func (dwr *DWRClient) tokenify(number int64) string {
	tokenbuf := []string{}
	charmap := []rune("1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ*$")
	remainder := number
	for remainder > 0 {
		tokenbuf = append(tokenbuf, string(charmap[remainder&0x3F]))
		remainder = int64(remainder / 64)
	}
	return strings.Join(tokenbuf, "")
}

func (dwr *DWRClient) setSession(dwrSessionID string) {
	url := &url.URL{
		Scheme: dwr.schema,
		Host:   dwr.host,
	}
	cookies := []*http.Cookie{
		{
			Name:  "DWRSESSIONID",
			Value: dwrSessionID,
		},
	}
	dwr.httpClient.Jar.SetCookies(url, cookies)

	pageID1 := dwr.tokenify(time.Now().UnixNano())
	pageID2 := dwr.tokenify(rand.Int63())
	pageID := fmt.Sprintf("%s-%s", pageID1, pageID2)

	dwr.scriptSessionID = fmt.Sprintf("%s/%s", dwrSessionID, pageID)
}

func (dwr *DWRClient) request(page, script, method string, args []string, extraParams map[string]string) (res *http.Response, err error) {
	if !dwr.initialized {
		return nil, fmt.Errorf("DWRClient not initialized")
	}

	params := make(map[string]string)
	params["page"] = strings.Replace(page, "/", "%2F", -1)
	params["batchId"] = strconv.Itoa(dwr.batchID)
	params["scriptSessionId"] = dwr.scriptSessionID
	params["c0-scriptName"] = script
	params["c0-methodName"] = method

	buf := bytes.Buffer{}
	for k, v := range params {
		buf.WriteString(fmt.Sprintf("%s=%s\n", k, v))
	}
	for k, v := range dwr.params {
		buf.WriteString(fmt.Sprintf("%s=%s\n", k, v))
	}
	for k, v := range extraParams {
		buf.WriteString(fmt.Sprintf("%s=%s\n", k, v))
	}

	url := dwr.host + "/dwr/call/plaincall/" + script + "." + method + ".dwr"

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, err
	}

	res, err = dwr.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	dwr.batchID++
	return res, nil
}
