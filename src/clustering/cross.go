package clustering

import (
	"bytes"
	"conf"
	"log"
	"message"
	"queue"
	"time"
)

const (
	COMM_TYPE_ACKNOWLEDGED = 155
	COMM_TYPE_FAILED       = 156
)

var cross_queue []*queue.GrandmaQueue

type Region struct {
	url string
}

func init() {
	region_list := conf.GetRegionList()

	if region_list == nil {
		cross_queue = nil
		return
	}

	cross_queue = make([]*queue.GrandmaQueue, region_list.Len())

	for i := 0; i < len(cross_queue); i++ {
		cross_queue[i] = queue.NewQueue()
	}
}

func pushToCross(msg *message.Obj) {
	if cross_queue == nil {
		return
	}

	for i := 0; i < len(cross_queue); i++ {
		cross_queue[i].PushMessage(msg)
	}
}

// Working...

func executeCrossQueue() {
	if cross_queue == nil {
		return
	}

	var msg = cross_queue.PopMessage()

	buffer := bytes.NewBuffer([]byte{COMM_TYPE_SCHEDULE})
	buffer.Write(msg.GetMessagePayload())

	node, err := sendRemote(buffer.Bytes(), 0)

	if err != nil {
		cluster_queue.PushFront(msg)
		go executeCrossQueue()
		return
	}

	log.Println("Waiting for response from slave")

	go executeCrossQueue()

	select {
	case <-node.channel:
		log.Println("Got schedule id")
		node.update(node.complexity+3, node.index)
		// Do something with schedule id
		// Save to db or log
		break
	case <-time.After(3 * time.Second):
		cluster_queue.PushFront(msg)
		break
	}
}
