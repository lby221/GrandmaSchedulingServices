package clustering

import (
	"container/heap"
	"errors"
	"log"
	"message"
	"schedule"
)

var (
	ERR_CONSUME_MESSAGE = []byte{140}
)

const (
	COMM_TYPE_CONSUMED  byte = 150
	COMM_TYPE_SCHEDULE  byte = 151
	COMM_TYPE_FINISHED  byte = 152
	COMM_TYPE_HEARTBEAT byte = 153
)

var (
	Err_Distribute_Timeout  = errors.New("Distribute to slave timeout")
	Err_Distribute_Internal = errors.New("Distribute to slave internal error")
)

func (nq NodeQueue) Less(i, j int) bool {
	RWLock.RLock()
	first := nq[i].complexity
	second := nq[j].complexity
	RWLock.RUnlock()

	return first < second
}

func (nq NodeQueue) Len() int {
	RWLock.RLock()
	length := len(nq)
	RWLock.RUnlock()
	return length
}

func (nq NodeQueue) Swap(i, j int) {
	RWLock.Lock()
	temp := nq[i]
	nq[i] = nq[j]
	nq[j] = temp
	nq[i].index = j
	nq[j].index = i
	RWLock.Unlock()
}

func (nq NodeQueue) Push(x interface{}) {
	return
}

func (nq NodeQueue) Pop() interface{} {
	return nil
}

func (n *Node) update(complexity uint, position int) {
	RWLock.Lock()
	n.complexity = complexity
	RWLock.Unlock()
	heap.Fix(slave_connections, position)
}

func createComplexityRanking() {
	heap.Init(slave_connections)
}

func distLevel1Calls(msg *message.Obj, node_index ...int) (int, error) {
	var node *Node = nil
	var index int = 0

	if len(node_index) > 0 {
		index = node_index[0]
	}

	if slave_connections.Len() > index {
		RWLock.RLock()
		node = slave_connections[index]
		RWLock.RUnlock()
	}

	log.Printf("Distribute level 1 at slave %d with address %s", index, node)
	master_complexity := master_connection.complexity

	if node == nil || node.closed || node.complexity > master_complexity {
		log.Println("Scheduled on master")
		s, err := schedule.NewSchedule(msg)
		master_connection.complexity = master_complexity + 3
		go func() {
			select {
			case <-s.Signal:
				master_connection.complexity = master_complexity - 3
				break
			}
		}()
		return s.Id, err
	} else {
		log.Println("Preparing to send to slave")
		pushToSend(msg)
		return 1, nil
	}

	return -1, Err_Distribute_Internal
}

func consumeLevel1Calls(payload []byte) {
	log.Println("comsuming message...")
	msg, err := message.NewMessageFromPayload(payload)
	if err != nil {
		sendMaster(ERR_CONSUME_MESSAGE)
	}

	_, err = schedule.NewSchedule(msg)
	if err != nil {
		sendMaster(ERR_CONSUME_MESSAGE)
	}
}

func distLevel2Calls(msg *message.Obj) (int, error) {
	s, err := schedule.NewSchedule(msg)
	master_complexity := master_connection.complexity
	master_connection.complexity = master_complexity + 3
	go func() {
		select {
		case <-s.Signal:
			master_connection.complexity = master_complexity + 3
			log.Println(master_connection.complexity)
			break
		}
	}()
	return s.Id, err
}
