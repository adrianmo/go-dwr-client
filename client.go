package dwr

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
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

// Params used for DWR requests.
type Params map[string]string

func (p *Params) buffer() *bytes.Buffer {
	buf := bytes.Buffer{}
	for k, v := range *p {
		buf.WriteString(fmt.Sprintf("%s=%s\n", k, v))
	}
	return &buf
}

func (p *Params) String() string {
	return p.buffer().String()
}

// A Client is a Direct Web Remoting (DWR) client that manages communication with
// DWR endpoints.
type Client struct {
	// HTTP client used to communicate with the DWR server.
	httpClient *http.Client

	// Base URL for DWR requests.
	baseURL *url.URL

	// Auto-incremental batch ID.
	batchID int

	// Whether or not the DWR client is initialized.
	initialized bool

	// Script Session ID
	scriptSessionID string

	// Base parameters included in each request.
	baseParams Params
}

// NewClient returns a new DWR client.
func NewClient(baseURL string, baseParams *Params) (*Client, error) {
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

	u, err := url.Parse(baseURL)
	if err != nil {
		log.Fatal(err)
	}

	c := &Client{
		httpClient:  httpClient,
		baseURL:     u,
		batchID:     0,
		initialized: false,
		baseParams:  *baseParams,
	}

	err = c.init()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// HTTPClient returns the underlying HTTP client used by the DWR client.
func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

// SessionID returns the current DWR session ID used by the client.
// Returns an empty string if the client has not been initialized.
func (c *Client) SessionID() string {
	return c.scriptSessionID
}

// init initializes the DWR client by trying to obtain the session ID.
func (c *Client) init() error {
	c.batchID = 0
	script := "__System"
	method := "generateId"
	// Need to set initialized to true at this point to allow making the init request
	c.initialized = true

	res, err := c.Request("", script, method, nil, nil)
	if err != nil {
		c.initialized = false
		return err
	}
	defer res.Body.Close()

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		c.initialized = false
		return err
	}
	bodyString := string(bodyBytes)

	re := regexp.MustCompile(`handleCallback\("\w+",\s*"\w+",\s*"(.+?)"\);`)
	s := re.FindStringSubmatch(bodyString)
	if s == nil || len(s) < 2 {
		c.initialized = false
		return fmt.Errorf("Could not find session ID in response body")
	}
	dwrSessionID := s[1]
	c.setSession(dwrSessionID)
	return nil
}

func (c *Client) tokenify(number int64) string {
	tokenbuf := []string{}
	charmap := []rune("1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ*$")
	remainder := number
	for remainder > 0 {
		tokenbuf = append(tokenbuf, string(charmap[remainder&0x3F]))
		remainder = int64(remainder / 64)
	}
	return strings.Join(tokenbuf, "")
}

func (c *Client) setSession(dwrSessionID string) {
	cookies := []*http.Cookie{
		{
			Name:  "DWRSESSIONID",
			Value: dwrSessionID,
		},
	}
	c.httpClient.Jar.SetCookies(c.baseURL, cookies)

	pageID1 := c.tokenify(time.Now().UnixNano())
	pageID2 := c.tokenify(rand.Int63())
	pageID := fmt.Sprintf("%s-%s", pageID1, pageID2)

	c.scriptSessionID = fmt.Sprintf("%s/%s", dwrSessionID, pageID)
}

// Request formats and sends a DWR request (HTTP Post request) with the
// given DWR configuration. The parameters provided in extraParams will
// override any base parameter if there the key is already present.
func (c *Client) Request(page, script, method string, args []string, extraParams *Params) (res *http.Response, err error) {
	if !c.initialized {
		return nil, fmt.Errorf("DWRClient not initialized")
	}

	params := Params{
		"page":            strings.Replace(page, "/", "%2F", -1),
		"batchId":         strconv.Itoa(c.batchID),
		"scriptSessionId": c.scriptSessionID,
		"c0-scriptName":   script,
		"c0-methodName":   method,
	}

	for k, v := range c.baseParams {
		params[k] = v
	}
	if extraParams != nil {
		for k, v := range *extraParams {
			params[k] = v
		}
	}

	url := c.baseURL.String() + "/dwr/call/plaincall/" + script + "." + method + ".dwr"

	req, err := http.NewRequest("POST", url, params.buffer())
	if err != nil {
		return nil, err
	}

	res, err = c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	c.batchID++
	return res, nil
}
