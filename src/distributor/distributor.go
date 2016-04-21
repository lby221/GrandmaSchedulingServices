package distributor

import (
	"message"
	"queue"
)

func ProcessMessageQueue() {
	var msg = queue.Main_Queue.PopMessage()

	var dist_type = msg.MessageType
	var endpoint = msg.Endpoint
	var msg_body = msg.MessageBody
	switch dist_type {
	case message.S_DELETE_MESSAGE:
		go ProcessMessageQueue()
	case message.S_REST_NOTIFICATION:
		go sendRESTCall(endpoint, msg_body)
		go ProcessMessageQueue()
	case message.S_WEBSOCKET_NOTIFICATION:
		go sendWebSocketMsg(endpoint, msg_body)
		go ProcessMessageQueue()
	case message.S_SMS_NOTIFICATION:
		go sendSMSMessage(endpoint, msg_body)
		go ProcessMessageQueue()
	case message.S_GCM_NOTIFICATION:
		go sendGCMPushNotification(endpoint, msg_body)
		go ProcessMessageQueue()
	// case message.S_APNS_NOTIFICATION:
	// 	go sendAPNSPushNotification(endpoint, msg_body)
	// 	go ProcessMessageQueue()
	// case message.S_TOPIC_NOTIFICATION:
	// 	go sendTopicPushNotification(endpoint, msg_body)
	// 	go ProcessMessageQueue()
	case message.S_EMAIL_NOTIFICATION:
		go sendEmailMsg(endpoint, msg_body)
		go ProcessMessageQueue()
	default:
		go ProcessMessageQueue()
	}
}
