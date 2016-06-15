package distributor

import (
	"bytes"
	"io/ioutil"
	"jsonwrapper"
	"log"
	"net/http"
)

func sendGCMCall(msg *string) error {
	req, err := http.NewRequest("POST", "https://gcm-http.googleapis.com/gcm/send", bytes.NewBufferString(*msg))

	if err != nil {
		return ErrorInvalidEndpointOrBody
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "key")
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

func sendGCMPushNotification(endpoint string, message string) error {
	msg, err := jsonwrapper.NewObjectFromBytes([]byte(message))
	if err != nil {
		return err
	}
	msg_type, err := msg.GetString("type")
	if err != nil {
		return err
	}
	msg_title, err := msg.GetString("title")
	if err != nil {
		return err
	}
	msg_body, err := msg.GetString("body")
	if err != nil {
		return err
	}
	msg_data, err := msg.GetObject("payload")
	if err != nil {
		return err
	}

	msg_to_send := `{
  		"to" : "/topics/` + endpoint + `",
  		"priority": "high",
  		"data" : {
  		"title": "` + msg_title + `",
    	"message" : "` + msg_body + `",
    	"payload": {"type": "` + msg_type + `", "data": ` + msg_data.String() + `}
  		},
  		"notification" : {
    	"title": "` + msg_title + `",
    	"body" : "` + msg_body + `",
    	"badge": "1",
    	"sound": "default",
    	"payload": {"type": "` + msg_type + `", "data": ` + msg_data.String() + `}
  		}
	}`

	log.Println(msg_to_send)

	err = sendGCMCall(&msg_to_send)

	if err != nil {
		return err
	}

	return nil
}

// func sendAPNSPushNotification(endpoint string, message string) error {
// 	msg := aws.String(`{"APNS":"{\"aps\":{\"alert\":` + message + `}}"}`)
// 	return sendPushNotification(endpoint, msg)
// }

// func sendBAIDUPushNotification(endpoint string, message string) error {
// 	msg := aws.String(`{"BAIDU":"{` + message + `}"}`)
// 	return sendPushNotification(endpoint, msg)
// }

// func sendTopicPushNotification(endpoint string, message string) error {
// 	msg := aws.String(`{"GCM":"{\"data\":{\"message\":` + message + `}}",
// 		"APNS":"{\"aps\":{\"alert\":` + message + `}}"}`)
// 	return sendPushNotification(endpoint, msg)
// }

// func sendPushNotification(endpoint string, message *string) error {
// 	sns_service := sns.New(session.New(), aws.NewConfig().WithRegion("us-west-1"))

// 	params := &sns.PublishInput{
// 		Message:          message,
// 		MessageStructure: aws.String("json"),
// 		MessageAttributes: map[string]*sns.MessageAttributeValue{
// 			"Powered-By": {
// 				DataType:    aws.String("String"),
// 				StringValue: aws.String("GrandmaSchedulerServices"),
// 			},
// 		},
// 		TargetArn: aws.String(endpoint),
// 	}
// 	_, err := sns_service.Publish(params)

// 	if err != nil {
// 		log.Println(err)
// 		return err
// 	}

// 	return nil
// }
