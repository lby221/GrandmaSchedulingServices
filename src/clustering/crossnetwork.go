package clustering

import (
	"conf"
	"net/http"
	"strconv"
)

var (
	ErrorInproperResponse = errors.New("Status code not 200")
)

func sendCross(msg, url string, msg_type int) error {
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(msg))

	req.Header.Add("Content-Type", content_type)
	req.Header.Add("X-Powered-By", "GrandmaSchedulingServices")
	req.Header.Add("X-Grandma-Relay-Type", strconv.Itoa(msg_type))
	req.Header.Add("X-Grandma-Relay", "true")

	client := &http.Client{}
	response, err := client.Do(req)

	if err != nil {
		return err
	}

	defer response.Body.Close()
	body, _ := ioutil.ReadAll(response.Body)

	log.Println(string(body))

	if response.StatusCode != 200 {
		return ErrorInproperResponse
	}

	return nil
}
