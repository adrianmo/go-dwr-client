package dwr

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	suite.Suite
	testServer *httptest.Server
}

// Make sure that VariableThatShouldStartAtFive is set to five
// before each test
func (suite *ClientTestSuite) SetupSuite() {
	suite.testServer = httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/dwr/call/plaincall/__System.generateId.dwr" {
				w.Write([]byte(`throw 'allowScriptTagRemoting is false.';
(function(){
var r=window.dwr._[1];
//#DWR-INSERT
//#DWR-REPLY
r.handleCallback("0","0","wfpAFVlyUnpW9EclMjKcxVXUr7n");
})();`))
			}
			if r.URL.Path == "/dwr/call/plaincall/MySvcAjax.getData.dwr" {
				w.Write([]byte(`throw 'allowScriptTagRemoting is false.';
(function(){
var r=window.dwr._[1];
//#DWR-INSERT
//#DWR-REPLY
r.handleCallback("0","0",{foo:"bar"});
})();`))
			}
		}),
	)
}

func (suite *ClientTestSuite) TearDownSuite() {
	suite.testServer.Close()
}

func (suite *ClientTestSuite) TestClient() {
	baseParams := &Params{
		"callCount":  "1",
		"windowName": "foo",
		"instanceId": "1",
		"c0-id":      "0",
	}

	client, err := NewClient(suite.testServer.URL, baseParams)
	suite.Nil(err)
	suite.Contains(client.SessionID(), "wfpAFVlyUnpW9EclMjKcxVXUr7n")

	extraParams := &Params{
		"c0-e1":     "string:87345",
		"c0-e2":     "string:X709183",
		"c0-param0": "Object_Object:{foo:reference:c0-e1, bar:reference:c0-e2}",
	}
	args := []string{"firstArg"}

	res, err := client.Request("info.do", "MySvcAjax", "getData", args, extraParams)
	suite.Nil(err)
	defer res.Body.Close()
	bodyBytes, _ := ioutil.ReadAll(res.Body)
	bodyString := string(bodyBytes)
	suite.Contains(bodyString, "{foo:\"bar\"}")
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}
