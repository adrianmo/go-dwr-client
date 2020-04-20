# go-dwr-client

go-dwr-client is a simple Direct Web Remoting Client (DWR) written in Go. This client can be used to 
communicate with DWR servers.

Beware that this project is a prototype, therefore expect bugs and breaking changes in the future.

## Usage

```go
package main

import (
	"log"

	"github.com/adrianmo/go-dwr-client"
)

func main() {
	baseParams := &dwr.Params{
		"callCount":  "1",
		"windowName": "foo",
		"instanceId": "1",
		"c0-id":      "0",
	}

	client, err := dwr.NewClient("https://example.com/vol", baseParams)
	if err != nil {
		log.Fatal(err)
	}

	extraParams := &dwr.Params{
		"c0-e1":     "string:87345",
		"c0-e2":     "string:X709183",
		"c0-param0": "Object_Object:{foo:reference:c0-e1, bar:reference:c0-e2}",
	}
	args := []string{"firstArg"}

	res, err := client.Request("/vol/info.do", "MySvcAjax", "getData", args, extraParams)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(res.Body)
}
```

## Contributing

Issues and Pull Requests are welcomed and encouraged.
