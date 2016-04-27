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

	REGION_STATUS_OK       = 130
	REGION_STATUS_DOWN     = 131
	REGION_STATUS_UNSTABLE = 132
)

var cross_nodes []*Region

type Region struct {
	Url          string
	Status       byte
	SendingQueue *queue.GrandmaQueue
}

func init() {
	region_list := conf.GetRegionList()

	if region_list == nil {
		cross_nodes = nil
		return
	}

	cross_nodes = make([]*queue.GrandmaQueue, region_list.Len())

	index := 0
	for e := region_list.Front(); e != nil; e = e.Next() {
		region := new(Region)
		region.SendingQueue = queue.NewQueue()
		region.Status = REGION_STATUS_OK
		region.Url = e.Value()
		cross_nodes[index] = region
	}
}

// Distribution message mapping
func preprocessMapping(config []byte) {

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
