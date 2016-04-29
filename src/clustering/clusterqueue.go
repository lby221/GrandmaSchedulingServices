package clustering

import (
	"bytes"
	"log"
	"message"
	"queue"
	"time"
)

var cluster_queue *queue.GrandmaQueue

func init() {
	cluster_queue = queue.NewQueue()
}

func pushToSend(msg *message.Obj) {
	cluster_queue.PushMessage(msg)
}

func executeSlaveQueue() {
	var msg = cluster_queue.PopMessage()

	buffer := bytes.NewBuffer([]byte{COMM_TYPE_SCHEDULE})
	buffer.Write(msg.GetMessagePayload())

	node, err := sendSlave(buffer.Bytes(), 0)

	if err != nil {
		cluster_queue.PushFront(msg)
		go executeSlaveQueue()
		return
	}

	log.Println("Waiting for response from slave")

	go executeSlaveQueue()

	select {
	case <-node.channel:
		log.Println("Got schedule id")
		node.update(node.complexity+3, node.index)
		// Do something with schedule id
		// Save to db or log
		break
	case <-time.After(3 * time.Second):
		log.Println("3 seconds bad")
		cluster_queue.PushFront(msg)
		break
	}
}
