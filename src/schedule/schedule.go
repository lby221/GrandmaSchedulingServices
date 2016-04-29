package schedule

import (
	"conf"
	"encoding/binary"
	"errors"
	"log"
	"message"
	"net"
	"queue"
	"strings"
	"sync"
	"time"
)

var dbConnection = "127.0.0.1:3306"
var dbUsername = "root"
var dbPassword = ""
var tcp *net.TCPConn = nil
var tcplock = new(sync.Mutex)

type Schedule struct {
	Id     int
	Exp    int64
	Signal chan bool
}

var (
	COMM_TYPE_CONSUMED byte = 150
	COMM_TYPE_SCHEDULE byte = 151
	COMM_TYPE_FINISHED byte = 152
)

var (
	ErrorFailedRecoveringFromHistory = errors.New("Failed to recover from crash")
	ErrorInvalidMessageContent       = errors.New("Invalid message content")
	ErrorInternalDBSettings          = errors.New("Invalid database settings")
)

func InitScheduler(c *net.TCPConn) {
	tcplock.Lock()
	tcp = c
	tcplock.Unlock()
	initializeMySQLDatabase()
	if err := recoverSchedule(); err != nil {
		panic(err)
	}
	startLoop()
}

func ChangeConnection(c *net.TCPConn) {
	log.Println("TCP connection changed")
	tcplock.Lock()
	tcp = c
	tcplock.Unlock()
}

func reportSchedule() {
	msg_size_header := make([]byte, 2)
	binary.BigEndian.PutUint16(msg_size_header, uint16(1))

	tcplock.Lock()
	_, err := tcp.Write(msg_size_header)
	_, err = tcp.Write([]byte{COMM_TYPE_FINISHED})
	tcplock.Unlock()

	if err != nil {
		log.Println("failed sending msg")
	}

	log.Println("reported result")
}

func consumeSchedule(id int32) {
	buffer := make([]byte, 4)
	binary.LittleEndian.PutUint32(buffer, uint32(id))

	to_send := make([]byte, 5)
	to_send[0] = COMM_TYPE_CONSUMED

	for i := 0; i < 4; i++ {
		to_send[i+1] = buffer[i]
	}

	msg_size_header := make([]byte, 2)
	binary.BigEndian.PutUint16(msg_size_header, uint16(5))
	tcplock.Lock()
	tcp.Write(msg_size_header)
	tcp.Write(to_send)
	tcplock.Unlock()
	log.Println("reported consume")
}

func recoverSchedule() error {
	log.Println("Recovering schedules...")

	conn, err := getMySQLConnector()
	if err != nil {
		panic(err)
	}

	rows, _, err := conn.Query("SELECT * FROM records_" +
		strings.Replace(conf.GetGrandmaName(), " ", "_", -1) + " WHERE sent = FALSE")
	if err != nil {
		return err
	}

	for i := 0; i < len(rows); i++ {
		row := rows[i]
		current_time := time.Now().UnixNano() / 1000000

		schedule_to_recover := Schedule{row.Int(0), row.Int64(4) - current_time, make(chan bool, 1)}

		log.Println(current_time)
		log.Println(row.Int64(4))

		if current_time+1000 > row.Int64(4) {
			schedule_to_recover.pushToSendingQueue()
		} else {
			err = put(&schedule_to_recover)
			if err != nil {
				panic(err)
			}
		}

		log.Printf("recovered: %d", schedule_to_recover.Id)
	}

	conn.Close()

	return nil

}

func NewSchedule(m *message.Obj) (*Schedule, error) {
	conn, err := getMySQLConnector()
	if err != nil {
		panic(err)
	}

	current_time := time.Now().UnixNano() / 1000000

	stmt, err := conn.Prepare("INSERT INTO records_" +
		strings.Replace(conf.GetGrandmaName(), " ", "_", -1) + " VALUES (NULL, ?, ?, ?, ?, FALSE, NULL)")
	if err != nil {
		return nil, ErrorInvalidMessageContent
	}

	stmt.Run(m.MessageType, m.Endpoint, m.MessageBody, current_time+m.Expiration)
	log.Printf("ttl: %d", current_time+m.Expiration)
	rows, _, err := conn.Query("SELECT LAST_INSERT_ID()")
	if err != nil {
		return nil, ErrorInternalDBSettings
	}
	conn.Close()

	var id = rows[0]
	var s = Schedule{id.Int(0), m.Expiration, make(chan bool, 1)}

	log.Print("Schedule id generated by slave: ")
	log.Println(id)

	if m.Expiration < 1000 {
		s.pushToSendingQueue()
		if tcp != nil {
			consumeSchedule(int32(id.Int(0)))
		}
		return &s, nil
	}

	err = put(&s)

	if err != nil {
		panic(err)
	}

	if tcp != nil {
		consumeSchedule(int32(id.Int(0)))
	}
	return &s, nil
}

func (s *Schedule) pushToSendingQueue() error {
	log.Println("push message...")

	conn, err := getMySQLConnector()
	if err != nil {
		panic(err)
	}

	rows, _, err := conn.Query("SELECT * FROM records_"+
		strings.Replace(conf.GetGrandmaName(), " ", "_", -1)+" WHERE id = %d", s.Id)
	if err != nil {
		return ErrorInternalDBSettings
	}

	stmt, err := conn.Prepare("UPDATE records_" +
		strings.Replace(conf.GetGrandmaName(), " ", "_", -1) + " SET sent = TRUE WHERE id = ?")
	if err != nil {
		return ErrorInternalDBSettings
	}
	stmt.Run(s.Id)

	conn.Close()

	row := rows[0]
	msg_to_push := new(message.Obj)

	log.Println(msg_to_push.Endpoint)

	msg_to_push.MessageType = row.Int(1)
	msg_to_push.Endpoint = row.Str(2)
	msg_to_push.MessageBody = row.Str(3)
	msg_to_push.Expiration = 0

	queue.Main_Queue.PushMessage(msg_to_push)

	if tcp == nil {
		s.Signal <- true
	} else {
		reportSchedule()
	}

	return nil
}
