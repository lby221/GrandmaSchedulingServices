package ws

import (
	"container/list"
	"errors"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

const (
	maxConn        = 20
	pongWait       = 15 * time.Second
	pingPeriod     = pongWait * 4 / 5
	writeWait      = 10 * time.Second
	maxMessageSize = 64
)

var (
	ErrorExceedsMaxConn   = errors.New("Exceeds max connections allowed")
	ErrorNoLiveConnection = errors.New("No connection under this id")
)

type connection struct {
	id   string
	key  string
	conn *websocket.Conn
	msg  chan []byte
}

var conn_map = make(map[string]*list.List)

func (c *connection) Join() error {
	id := c.id
	if conn_map[id] == nil {
		conn_map[id] = list.New()
	}

	if conn_map[id].Len() > maxConn {
		return ErrorExceedsMaxConn
	}

	conn_map[id].PushBack(c)
	return nil
}

func (c *connection) Leave() error {
	id := c.id
	if conn_map[id] == nil {
		return ErrorNoLiveConnection
	}

	p := conn_map[id]
	for e := p.Front(); e != nil; e = e.Next() {
		if e.Value == c {
			p.Remove(e)
			if p.Len() == 0 {
				delete(conn_map, id)
			}
			return nil
		}
	}

	return ErrorNoLiveConnection
}

func NewWs(c *websocket.Conn, id string, key string) *connection {
	conn := new(connection)
	conn.id = id
	conn.key = key
	conn.conn = c
	conn.msg = make(chan []byte)

	return conn
}

func Send(id string, key string, msg string) error {
	if conn_map[id] == nil {
		log.Println("id not found")
		return ErrorNoLiveConnection
	}

	log.Println("wsmessage sent")

	for e := conn_map[id].Front(); e != nil; e = e.Next() {
		c := e.Value.(*connection)

		if key != "" {
			if c.key == key {
				c.msg <- []byte(msg)
			}
		} else {
			c.msg <- []byte(msg)
		}
	}

	return nil
}

func (c *connection) write(mt int, payload []byte) error {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(mt, payload)
}

func (c *connection) Heartbeat() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		log.Println("see you")
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.msg:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (c *connection) GetHeartbeat() {
	defer func() {
		c.Leave()
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
