package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/burdzwastaken/concourse-spinnaker-resource/concourse"
)

var (
	// TriggerParams byte array
	TriggerParams []byte
	// Params string
	Params string
	// Data interface map
	Data map[string]interface{}
)

func main() {
	var request concourse.OutRequest

	concourse.ReadRequest(&request)
	tr := &http.Transport{}
	if request.Source.X509Cert != "" {
		cert, err := tls.X509KeyPair([]byte(request.Source.X509Cert), []byte(request.Source.X509Key))
		if err != nil {
			concourse.Fatal("Error reading X509 key pair: \n%v\n", err)
		}

		tlsConfig := &tls.Config{
			MinVersion:               tls.VersionTLS12,
			PreferServerCipherSuites: true,
			Certificates:             []tls.Certificate{cert},
			InsecureSkipVerify:       true,
		}

		tr.TLSClientConfig = tlsConfig
	}

	client := &http.Client{Transport: tr}

	url := fmt.Sprintf("%s/pipelines/%s/%s", request.Source.SpinnakerAPI, request.Params.SpinnakerApplication, request.Params.SpinnakerPipeline)

	if len(request.Params.TriggerParams) == 0 {
		TriggerParams = []byte(`{"type": "concourse-resource"}`)
	} else {
		for key, value := range request.Params.TriggerParams {
			Params = Params + fmt.Sprintf("\"%s\":\"%s\",", key, value)
		}
		Params = strings.TrimSuffix(Params, ",")
		TriggerParams = []byte(`{"type": "concourse-resource", "parameters": {` + Params + `}}`)
	}

	concourse.Sayf("Executing pipeline: '%s'\n", url)

	output := concourse.OutResponse{}
	if spinnaker, err := client.Post(url, "application/json", bytes.NewBuffer(TriggerParams)); err != nil {
		concourse.Fatal("Unable to start pipeline because:\n%v\n", err)
	} else {
		body, err := ioutil.ReadAll(spinnaker.Body)
		err = json.Unmarshal([]byte(body), &Data)
		if err != nil {
			concourse.Fatal("Unable to parse JSON response because:\n%v\n", err)
		}
		output.Version = concourse.Version{
			ExecutionID: strings.Split(Data["ref"].(string), "/")[2],
		}
	}

	concourse.Sayf("Pipeline executed successfully")

	concourse.WriteResponse(output)
}
