package distributor

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

var (
	ErrorInvalidEndpointOrBody = errors.New("Invalid endpoint or body")
	ErrorNetworkDisconnect     = errors.New("Network error")
	ErrorInproperResponse      = errors.New("Status code not 200")
)

func sendRESTCall(endpoint string, msg string) error {
	endpoint_components := strings.Split(endpoint, " ")

	log.Println(endpoint)

	if len(endpoint_components) != 3 {
		log.Println("endpoint parsing error")
		return ErrorInvalidEndpointOrBody
	}

	method := endpoint_components[0]
	url := endpoint_components[1]
	content_type := endpoint_components[2]
	req, err := http.NewRequest(method, url, bytes.NewBufferString(msg))

	log.Println(url)

	if err != nil {
		return ErrorInvalidEndpointOrBody
	}

	req.Header.Add("Content-Type", content_type)
	req.Header.Add("Powered-By", "GrandmaSchedulerServices")

	client := &http.Client{}
	response, err := client.Do(req)

	if err != nil {
		return ErrorNetworkDisconnect
	}

	defer response.Body.Close()
	body, _ := ioutil.ReadAll(response.Body)

	log.Println(string(body))

	if response.StatusCode != 200 {
		return ErrorInproperResponse
	}

	return nil
}
