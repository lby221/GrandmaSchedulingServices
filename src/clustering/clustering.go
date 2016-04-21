package clustering

import (
	"conf"
	"fmt"
	"message"
	"net"
	"sync"
)

func DistCalls(msg *message.Obj) (int, error) {
	if msg.MessageType != 105 {
		return distLevel1Calls(msg)
	} else {
		return distLevel2Calls(msg)
	}
}

func Network() (bool, *net.TCPConn) {
	mode := conf.GetClusterMode()

	if mode {
		if conf.GetSlaveList() != nil {
			fmt.Println("Discovering slaves...")
			err, conns := discoverSlaves()
			if err != nil {
				fmt.Println("Failed connecting slaves")
				return false, nil
			}
			RWLock.Lock()
			slave_connections = make(NodeQueue, conns.Len())
			i := 0
			for e := conns.Front(); e != nil; e = e.Next() {
				slave_connections[i] = e.Value.(*Node)
				i++
			}
			master_connection = &Node{320, 320, 0, false, "", nil, make([]byte, 4096),
				make([]byte, 4096), 0, new(sync.RWMutex), nil}
			RWLock.Unlock()
			createComplexityRanking()
			rediscoverSlaves()
			go executeSlaveQueue()
			startLoopListener()
			return true, nil
		} else {
			err := waitMasterConnection()
			if err != nil {
				fmt.Println("\tSlave: Failed connecting master")
				return false, nil
			}
			rediscoverMasterConnection()
			startPointListener()
			return true, master_connection.conn
		}
	} else {
		master_connection = &Node{320, 320, 0, false, "", nil, nil,
			nil, 0, nil, nil}
		fmt.Println("\tSetting as single node mode...")
		return true, nil
	}
}
