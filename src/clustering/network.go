package clustering

import (
	"bytes"
	"conf"
	"container/list"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"schedule"
	//"net/http/httputil"
	"strconv"
	"sync"
	"time"
)

var (
	HANDSHAKE_L1_REQUEST = []byte{100}

	HANDSHAKE_L1_RESPONSE_OK      = []byte{200}
	HANDSHAKE_L1_RESPONSE_AUTH    = []byte{201}
	HANDSHAKE_L1_RESPONSE_BAD     = []byte{202}
	HANDSHAKE_L1_RESPONSE_REFUSE  = []byte{203}
	HANDSHAKE_L1_RESPONSE_TIMEOUT = []byte{204}

	HANDSHAKE_L2_RESPONSE_OK      = []byte{210}
	HANDSHAKE_L2_RESPONSE_BAD     = []byte{211}
	HANDSHAKE_L2_RESPONSE_REFUSE  = []byte{212}
	HANDSHAKE_L2_RESPONSE_TIMEOUT = []byte{213}

	HANDSHAKE_L3_RESPONSE_OK      = []byte{220}
	HANDSHAKE_L3_RESPONSE_BAD     = []byte{221}
	HANDSHAKE_L3_RESPONSE_TIMEOUT = []byte{222}
)

const (
	HANDSHAKE_STATUS_SUCCESS int8 = 1
	HANDSHAKE_STATUS_TIMEOUT int8 = -2
	HANDSHAKE_STATUS_REFUSED int8 = -5
	HANDSHAKE_STATUS_BADCONN int8 = -1
	HANDSHAKE_STATUS_SVR_ERR int8 = -3
	HANDSHAKE_STATUS_UNKNOWN int8 = -4
)

var (
	Err_Handshake_Status_Timeout = errors.New("time out")
	Err_Handshake_Status_Refused = errors.New("connection refused")
	Err_Handshake_Status_Badconn = errors.New("bad connection")
	Err_Handshake_Status_Svr_Err = errors.New("internal server error")
	Err_Handshake_Status_Unknown = errors.New("unknown error")

	Err_No_Available_Slave = errors.New("No slave node connected")

	Err_Slave_Disconnected  = errors.New("Slave disconnected")
	Err_Master_Disconnected = errors.New("Slave disconnected")
)

type Node struct {
	complexity          uint
	saved               uint
	index               int
	closed              bool
	address             string
	conn                *net.TCPConn
	read_buffer         []byte
	level_2_buffer      []byte
	level_2_buffer_size int
	bufferLock          *sync.RWMutex
	channel             chan int
}

type NodeQueue []*Node

var slave_connections NodeQueue = nil
var master_connection *Node = nil
var dropped_connections *list.List = nil
var RWLock = new(sync.RWMutex)

func discoverSlaves() (error, *list.List) {
	count := 1
	slave_list := conf.GetSlaveList()
	nslaves := slave_list.Len()
	var connection_pool = list.New()
	for e := slave_list.Front(); e != nil; e = e.Next() {
		fmt.Println("\tMaster: Connecting slave " + strconv.Itoa(count) + " of " + strconv.Itoa(nslaves))
		tcp, err := net.ResolveTCPAddr("tcp", e.Value.(string))
		if err != nil {
			return err, nil
		}

		tcp_conn, err := net.DialTCP("tcp", nil, tcp)

		if err != nil {
			return err, nil
		}

		tcp_conn.SetWriteBuffer(4096)
		tcp_conn.SetReadBuffer(64)

		status := handshakeMaster(tcp_conn)

		switch status {
		case HANDSHAKE_STATUS_BADCONN:
			fmt.Println("\tMaster: " + Err_Handshake_Status_Badconn.Error())
			continue
		case HANDSHAKE_STATUS_REFUSED:
			fmt.Println("\tMaster: " + Err_Handshake_Status_Refused.Error())
			continue
		case HANDSHAKE_STATUS_TIMEOUT:
			fmt.Println("\tMaster: " + Err_Handshake_Status_Timeout.Error())
			continue
		case HANDSHAKE_STATUS_UNKNOWN:
			fmt.Println("\tMaster: " + Err_Handshake_Status_Unknown.Error())
			continue
		}

		if status == HANDSHAKE_STATUS_SUCCESS {
			err := tcp_conn.SetKeepAlive(true)
			if err != nil {
				return err, nil
			}
			connection_pool.PushBack(&Node{100, 100, count - 1, false, e.Value.(string), tcp_conn,
				make([]byte, 4096), make([]byte, 4096), 0, new(sync.RWMutex), make(chan int, 1)})
		}

		count++

	}

	if connection_pool.Len() < 1 {
		return Err_No_Available_Slave, nil
	}

	return nil, connection_pool
}

func rediscoverSlaves() {
	dropped_connections = list.New()
	go func() {
		// Time for handshake with all slaves
		time.Sleep(time.Second * 30)
		log.Println("Rediscovering loop")
		for {
			heartbeatSlave()
			if dropped_connections.Len() > 0 {
				for n := dropped_connections.Front(); n != nil; n = n.Next() {
					node := n.Value.(*Node)

					if node.closed != true {
						RWLock.Lock()
						dropped_connections.Remove(n)
						RWLock.Unlock()
						continue
					}

					tcp, err := net.ResolveTCPAddr("tcp", node.address)
					if err != nil {
						continue
					}

					tcp_conn, err := net.DialTCP("tcp", nil, tcp)

					if err != nil {
						continue
					}

					tcp_conn.SetWriteBuffer(4096)
					tcp_conn.SetReadBuffer(64)

					status := handshakeMaster(tcp_conn)

					switch status {
					case HANDSHAKE_STATUS_BADCONN:
						fmt.Println("\tMaster: " + Err_Handshake_Status_Badconn.Error())
						continue
					case HANDSHAKE_STATUS_REFUSED:
						fmt.Println("\tMaster: " + Err_Handshake_Status_Refused.Error())
						continue
					case HANDSHAKE_STATUS_TIMEOUT:
						fmt.Println("\tMaster: " + Err_Handshake_Status_Timeout.Error())
						continue
					case HANDSHAKE_STATUS_UNKNOWN:
						fmt.Println("\tMaster: " + Err_Handshake_Status_Unknown.Error())
						continue
					}

					if status == HANDSHAKE_STATUS_SUCCESS {
						err := tcp_conn.SetKeepAlive(true)
						if err != nil {
							continue
						}
						RWLock.Lock()
						node.conn = tcp_conn
						node.closed = false
						dropped_connections.Remove(n)
						RWLock.Unlock()
						node.update(node.saved, node.index)
					}
				}
			}
			time.Sleep(5 * time.Second)
		}
	}()
}

func heartbeatSlave() {
	for index, _ := range slave_connections {
		sendSlave(bytes.NewBuffer([]byte{COMM_TYPE_HEARTBEAT}).Bytes(), index)
	}
}

func heartbeatMaster() {
	sendMaster(bytes.NewBuffer([]byte{COMM_TYPE_HEARTBEAT}).Bytes())
}

func waitMasterConnection() error {
	port := conf.GetNetworkPort()

	tcp, err := net.ResolveTCPAddr("tcp", port)
	if err != nil {
		return err
	}
	tcp_conn, err := net.ListenTCP("tcp", tcp)
	if err != nil {
		return err
	}

	for {
		fmt.Println("\tSlave: Waiting for master...")

		session, err := tcp_conn.AcceptTCP()
		if err != nil {
			return err
		}
		session.SetWriteBuffer(64)
		session.SetReadBuffer(4096)
		status := handshakeSlave(session)

		switch status {
		case HANDSHAKE_STATUS_BADCONN:
			fmt.Println("\tSlave: " + Err_Handshake_Status_Badconn.Error())
			continue
		case HANDSHAKE_STATUS_REFUSED:
			fmt.Println("\tSlave: " + Err_Handshake_Status_Refused.Error())
			continue
		case HANDSHAKE_STATUS_TIMEOUT:
			fmt.Println("\tSlave: " + Err_Handshake_Status_Timeout.Error())
			continue
		case HANDSHAKE_STATUS_UNKNOWN:
			fmt.Println("\tSlave: " + Err_Handshake_Status_Unknown.Error())
			continue
		}

		if status == HANDSHAKE_STATUS_SUCCESS {
			err := session.SetKeepAlive(true)
			if err != nil {
				return err
			}
			master_connection = &Node{320, 320, 0, false, "", session, make([]byte, 4096),
				make([]byte, 4096), 0, new(sync.RWMutex), nil}
			break
		}
	}

	return nil

}

func rediscoverMasterConnection() {
	go func() {
		// Time for handshake with master
		time.Sleep(time.Second * 30)
		log.Println("Rediscovering loop")
		for {
			heartbeatMaster()
			if master_connection.closed {
				port := conf.GetNetworkPort()

				tcp, err := net.ResolveTCPAddr("tcp", port)
				if err != nil {
					continue
				}
				tcp_conn, err := net.ListenTCP("tcp", tcp)
				if err != nil {
					continue
				}

				for {
					fmt.Println("\tSlave: Waiting for master...")

					session, err := tcp_conn.AcceptTCP()
					if err != nil {
						continue
					}
					session.SetWriteBuffer(64)
					session.SetReadBuffer(4096)
					status := handshakeSlave(session)

					switch status {
					case HANDSHAKE_STATUS_BADCONN:
						fmt.Println("\tSlave: " + Err_Handshake_Status_Badconn.Error())
						continue
					case HANDSHAKE_STATUS_REFUSED:
						fmt.Println("\tSlave: " + Err_Handshake_Status_Refused.Error())
						continue
					case HANDSHAKE_STATUS_TIMEOUT:
						fmt.Println("\tSlave: " + Err_Handshake_Status_Timeout.Error())
						continue
					case HANDSHAKE_STATUS_UNKNOWN:
						fmt.Println("\tSlave: " + Err_Handshake_Status_Unknown.Error())
						continue
					}

					if status == HANDSHAKE_STATUS_SUCCESS {
						err := session.SetKeepAlive(true)
						if err != nil {
							continue
						}
						RWLock.Lock()
						master_connection.conn = session
						master_connection.closed = false
						RWLock.Unlock()
						schedule.ChangeConnection(session)
						break
					}
				}
			}
			time.Sleep(1 * time.Second)
		}
	}()
}

func handshakeMaster(conn *net.TCPConn) int8 {
	_, err := conn.Write(HANDSHAKE_L1_REQUEST)
	if err != nil {
		return HANDSHAKE_STATUS_SVR_ERR
	}

	code, data := readAndCheckTimeOut(conn)
	if code != 0 {
		return code
	}

	if bytes.Compare(data, HANDSHAKE_L1_RESPONSE_AUTH) == 0 {
		fmt.Println("\tMaster: Sending authentication secret...")
		_, err := conn.Write([]byte(conf.GetNetworkSecret()))
		if err != nil {
			return HANDSHAKE_STATUS_SVR_ERR
		}
		code, data := readAndCheckTimeOut(conn)
		if code != 0 {
			return code
		}

		if bytes.Compare(data, HANDSHAKE_L2_RESPONSE_OK) != 0 {
			return HANDSHAKE_STATUS_REFUSED
		}
	} else if bytes.Compare(data, HANDSHAKE_L1_RESPONSE_OK) == 0 {

	} else {
		return HANDSHAKE_STATUS_UNKNOWN // more work for other responses
	}

	_, err = conn.Write([]byte(conf.GetGrandmaName()))
	if err != nil {
		return HANDSHAKE_STATUS_SVR_ERR
	}

	code, data = readAndCheckTimeOut(conn)
	if code == HANDSHAKE_STATUS_BADCONN {
		conn.Write(HANDSHAKE_L3_RESPONSE_BAD)
		return HANDSHAKE_STATUS_BADCONN
	} else if code == HANDSHAKE_STATUS_TIMEOUT {
		conn.Write(HANDSHAKE_L3_RESPONSE_TIMEOUT)
		return HANDSHAKE_STATUS_TIMEOUT
	}

	slave_name := string(data)
	fmt.Println("\tMaster: Connected to slave " + slave_name)

	_, err = conn.Write(HANDSHAKE_L3_RESPONSE_OK)
	if err != nil {
		return HANDSHAKE_STATUS_SVR_ERR
	}

	return HANDSHAKE_STATUS_SUCCESS
}

func handshakeSlave(conn *net.TCPConn) int8 {
	//var data []byte

	code, data := readAndCheckTimeOut(conn)
	if code == HANDSHAKE_STATUS_BADCONN {
		conn.Write(HANDSHAKE_L1_RESPONSE_BAD)
		return HANDSHAKE_STATUS_BADCONN
	} else if code == HANDSHAKE_STATUS_TIMEOUT {
		conn.Write(HANDSHAKE_L1_RESPONSE_TIMEOUT)
		return HANDSHAKE_STATUS_TIMEOUT
	}

	if bytes.Compare(data, HANDSHAKE_L1_REQUEST) != 0 {
		log.Println("l1 refuse slave")
		conn.Write(HANDSHAKE_L1_RESPONSE_REFUSE)
		return HANDSHAKE_STATUS_BADCONN
	}

	if conf.GetNetworkSecret() != "" {
		fmt.Println("\tSlave: Authenticating master...")
		_, err := conn.Write(HANDSHAKE_L1_RESPONSE_AUTH)
		if err != nil {
			conn.Write(HANDSHAKE_L1_RESPONSE_BAD)
			return HANDSHAKE_STATUS_SVR_ERR
		}

		code, data := readAndCheckTimeOut(conn)
		if code == HANDSHAKE_STATUS_BADCONN {
			conn.Write(HANDSHAKE_L2_RESPONSE_BAD)
			return HANDSHAKE_STATUS_BADCONN
		} else if code == HANDSHAKE_STATUS_TIMEOUT {
			conn.Write(HANDSHAKE_L2_RESPONSE_TIMEOUT)
			return HANDSHAKE_STATUS_TIMEOUT
		}

		if bytes.Compare(data, []byte(conf.GetNetworkSecret())) != 0 {
			conn.Write(HANDSHAKE_L2_RESPONSE_REFUSE)
			return HANDSHAKE_STATUS_BADCONN
		}

		_, err = conn.Write(HANDSHAKE_L2_RESPONSE_OK)
		if err != nil {
			conn.Write(HANDSHAKE_L2_RESPONSE_BAD)
			return HANDSHAKE_STATUS_SVR_ERR
		}
	} else {
		_, err := conn.Write(HANDSHAKE_L1_RESPONSE_OK)
		if err != nil {
			conn.Write(HANDSHAKE_L1_RESPONSE_BAD)
			return HANDSHAKE_STATUS_SVR_ERR
		}
	}

	code, data = readAndCheckTimeOut(conn)
	if code == HANDSHAKE_STATUS_BADCONN {
		conn.Write(HANDSHAKE_L3_RESPONSE_BAD)
		return HANDSHAKE_STATUS_BADCONN
	} else if code == HANDSHAKE_STATUS_TIMEOUT {
		conn.Write(HANDSHAKE_L3_RESPONSE_TIMEOUT)
		return HANDSHAKE_STATUS_TIMEOUT
	}

	master_name := string(data)
	fmt.Println("\tSlave: Connected to master " + master_name)

	_, err := conn.Write([]byte(conf.GetGrandmaName()))
	if err != nil {
		conn.Write(HANDSHAKE_L1_RESPONSE_BAD)
		return HANDSHAKE_STATUS_SVR_ERR
	}

	code, data = readAndCheckTimeOut(conn)
	if code != 0 {
		return code
	}

	if bytes.Compare(data, HANDSHAKE_L3_RESPONSE_OK) != 0 {
		return HANDSHAKE_STATUS_REFUSED
	}

	return HANDSHAKE_STATUS_SUCCESS

}

func readData(node *Node) uint {
	if node == nil || node.conn == nil {
		return 0
	}

	temp_buffer := make([]byte, 4096)
	var size int = 0
	var err error = nil

	if node.level_2_buffer_size > 1 {
		// Data available in level 2 buffer
		append_buffer(temp_buffer, node.level_2_buffer, 0, 0, node.level_2_buffer_size)
		size = node.level_2_buffer_size
		// Clear level 2 buffer
		node.level_2_buffer_size = 0
	} else if node.level_2_buffer_size == 1 {
		// Partial data available in level 2 buffer
		append_buffer(temp_buffer, node.level_2_buffer, 0, 0, node.level_2_buffer_size)
		size = node.level_2_buffer_size
		// Clear level 2 buffer
		node.level_2_buffer_size = 0
		// Continue reading data
		size, err = node.conn.Read(temp_buffer[1:])
		if err != nil {
			return 0
		}

		size = size + 1
	} else {
		// No level 2 buffer data available, read directly from socket
		if node.conn == nil {
			return 0
		}
		size, err = node.conn.Read(temp_buffer)
		if err != nil {
			return 0
		}
	}

	sofar := size - 2
	// bufferLock := node.bufferLock

	// log.Println("Reading data in progress...")
	// Consume all bytes in temp_buffer before continue
	// Read first 2 bytes and get message size
	var msg_size int16 = 0
	msg_size = ((msg_size | int16(temp_buffer[0])) << 8) | int16(temp_buffer[1])
	var msg_size_int int = int(msg_size)
	// log.Printf("Size of data: %d", size-2)
	// log.Printf("Size of message: %d", msg_size_int)
	if msg_size_int < 0 {
		return 0
	}
	// Copy rest to buffer
	if msg_size_int <= size-2 {
		// bufferLock.Lock()
		append_buffer(node.read_buffer, temp_buffer, 0, 2, msg_size_int)
		append_buffer(node.level_2_buffer, temp_buffer, 0, msg_size_int+2, size-2-msg_size_int)
		node.level_2_buffer_size = size - 2 - msg_size_int
		// bufferLock.Unlock()
		return uint(msg_size_int)
	}
	// bufferLock.Lock()
	append_buffer(node.read_buffer, temp_buffer, 0, 2, size-2)
	// bufferLock.Unlock()

	for {
		log.Println("Long data to read")

		bytes_to_read := msg_size_int - sofar

		if node.conn == nil {
			return 0
		}
		size, err = node.conn.Read(temp_buffer)
		if err != nil {
			return 0
		}

		current := sofar + size
		if current >= msg_size_int {
			// Save rest to level 2 buffer

			// bufferLock.Lock()
			append_buffer(node.level_2_buffer, temp_buffer, 0, bytes_to_read, current-msg_size_int)
			node.level_2_buffer_size = current - msg_size_int
			// Save data for this packet to buffer
			append_buffer(node.read_buffer, temp_buffer, sofar, 0, bytes_to_read)
			// bufferLock.Unlock()
			sofar = msg_size_int
			break
		} else {
			// Append message if more in buffer
			// bufferLock.Lock()
			append_buffer(node.read_buffer, temp_buffer, sofar, 0, size)
			// bufferLock.Unlock()
			sofar = current
		}
	}

	return uint(sofar)
}

func readHandshakeData(conn *net.TCPConn, buffer []byte) <-chan uint {
	out := make(chan uint, 1)
	go func() {
		temp_buffer := make([]byte, 512)
		size, err := conn.Read(temp_buffer)
		if err != nil {
			out <- 0
			close(out)
			return
		}
		append_buffer(buffer, temp_buffer, 0, 0, size)

		out <- uint(size)
		close(out)
	}()
	return out
}

func readAndCheckTimeOut(conn *net.TCPConn) (int8, []byte) {
	var ret []byte = nil
	buff := make([]byte, 64)
	select {
	case res := <-readHandshakeData(conn, buff):
		if res == 0 {
			return -1, ret
		}
		ret = make([]byte, res)
		copy(ret, buff)
	case <-time.After(5 * time.Second):
		return -2, ret
	}

	return 0, ret
}

func sendSlave(data []byte, slave_num int) (*Node, error) {
	RWLock.RLock()
	slave := slave_connections[slave_num]
	RWLock.RUnlock()

	if slave.closed {
		return slave, nil
	}

	slave_conn := slave.conn

	msg_size := int16(len(data))
	msg_size_header := make([]byte, 2)
	binary.BigEndian.PutUint16(msg_size_header, uint16(msg_size))

	_, err := slave_conn.Write(msg_size_header)
	_, err = slave_conn.Write(data)

	if err != nil {
		log.Println("Connection failed")
		disconnectSlave(slave_num)
		return slave, Err_Slave_Disconnected
	}
	return slave, nil
}

func sendMaster(data []byte) {
	if master_connection.closed {
		return
	}

	msg_size := int16(len(data))
	msg_size_header := make([]byte, 2)
	binary.BigEndian.PutUint16(msg_size_header, uint16(msg_size))
	RWLock.RLock()
	_, err := master_connection.conn.Write(msg_size_header)
	_, err = master_connection.conn.Write(data)
	RWLock.RUnlock()

	if err != nil {
		log.Println("Connection failed")
		disconnectMaster()
	}
}

func disconnectSlave(slave_num int) {

	RWLock.Lock()
	node := slave_connections[slave_num]
	node.conn.Close()

	node.closed = true
	node.saved = node.complexity
	RWLock.Unlock()

	node.update(999999, slave_num)
	dropped_connections.PushBack(node)

	log.Printf("slave number %d has been disconnected, %d nodes is not connected", slave_num, dropped_connections.Len())
}

func disconnectMaster() {
	RWLock.Lock()
	master_connection.closed = true
	master_connection.conn.Close()
	RWLock.Unlock()
}

func masterHandler(data []byte, node *Node) {
	buffer := bytes.NewReader(data)

	data_type, err := buffer.ReadByte()

	if err != nil {
		log.Println("Read byte error")
		return
	}

	switch data_type {
	case COMM_TYPE_CONSUMED:
		log.Println("slave job comsumed")
		var schedule_id int32
		err := binary.Read(buffer, binary.LittleEndian, &schedule_id)
		if err != nil {
			log.Println("Error reading schedule id")
			return
		}
		node.channel <- int(schedule_id)
		break
	case COMM_TYPE_FINISHED:
		log.Println("slave job finished")
		node.update(node.complexity-3, node.index)
		break
	case COMM_TYPE_HEARTBEAT:
		break
	default:
		log.Println("unknown type")
	}
}

func slaveHandler(data []byte) {
	buffer := bytes.NewReader(data)
	data_type, err := buffer.ReadByte()

	if err != nil {
		return
	}

	switch data_type {
	case COMM_TYPE_SCHEDULE:
		log.Println("received from master fc")
		byte_data := make([]byte, buffer.Len())
		buffer.Read(byte_data)
		consumeLevel1Calls(byte_data)
		break
	case COMM_TYPE_HEARTBEAT:
		break
	}
}

func startLoopListener() {
	var nslaves int = len(slave_connections)
	log.Println(nslaves)

	for i := 0; i < nslaves; i++ {
		node := slave_connections[i]
		go func(n *Node) {
			for {
				size := readData(n)
				if size > 0 {
					log.Println(size)
					// node.bufferLock.RLock()
					data := make([]byte, size)
					copy(data, n.read_buffer)
					go masterHandler(data, n)
				}
			}
		}(node)
	}
}

func startPointListener() {
	go func() {
		for {
			size := readData(master_connection)
			if size > 0 {
				data := make([]byte, size)
				// master_connection.bufferLock.RLock()
				copy(data, master_connection.read_buffer)
				// master_connection.bufferLock.Unlock()
				go slaveHandler(data)
			}
		}
	}()
}

// func setupProxyServer() {

// 	httputil.NewSingleHostReverseProxy(target)
// }
