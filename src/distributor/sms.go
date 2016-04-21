package distributor

import (
	"net/http"
	"net/url"
	"strings"
)

func sendSMSMessage(endpoint string, msg string) error {
	// Set initial variables
	account_sid := "ACdd27abf6914ae131ad2248a529eff4aa"
	auth_token := "9e2f645e33b1ac624a84deea89cebcc6"
	url_str := "https://api.twilio.com/2010-04-01/Accounts/" + account_sid + "/Messages.json"

	// Build out the data for our message
	v := url.Values{}
	v.Set("To", endpoint)
	v.Set("From", "+14152149780")
	v.Set("Body", msg)
	rb := *strings.NewReader(v.Encode())

	// Create client
	client := &http.Client{}

	req, err := http.NewRequest("POST", url_str, &rb)

	if err != nil {
		return ErrorNetworkDisconnect
	}

	req.SetBasicAuth(account_sid, auth_token)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Make request
	response, _ := client.Do(req)

	if response.StatusCode != 200 {
		return ErrorInproperResponse
	}

	return nil
}
