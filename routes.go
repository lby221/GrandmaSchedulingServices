package main

import (
	"clustering"
	"fmt"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"jsonwrapper"
	"log"
	"message"
	"net/http"
	"signature"
	// "ws"
	//"./users"     // According to your OAuth settings
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(*http.Request) bool { return true },
}

// Websocket only works with your oauth set up
/*
func handlerWs(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Powered-By", "GrandmaSchedulerServices")

	token := r.URL.Query().Get("key")

	res := users.GetUser(token)

	if res == nil {
		fmt.Fprintf(w, `{"failure":{"msg":"Not authorized"}}`)
		return
	}

	id, err := res.GetString("id")

	if err != nil {
		fmt.Fprintf(w, `{"failure":{"msg":"Internal error"}}`)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		fmt.Fprintf(w, `{"failure":{"msg":"`+err.Error()+`"}}`)
		return
	}

	wsconn := ws.NewWs(conn, id, token)
	wsconn.Join()

	go wsconn.Heartbeat()
	go wsconn.GetHeartbeat()

	return
}
*/

func handler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Powered-By", "GrandmaSchedulerServices")

	if r.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"failure":{"msg":"Bad request"}}`)
		return
	}

	time := r.URL.Query().Get("time")
	token := r.URL.Query().Get("token")

	body, err := ioutil.ReadAll(r.Body)
	log.Println(time)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"failure":{"msg":"Bad request"}}`)
		return
	}

	to_compare := signature.VToken("POST", time, string(body))
	//to_compare = "test"

	if token != to_compare {
		log.Println("not authorized")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, `{"failure":{"msg":"Not authorized"}}`)
		return
	} else {
		json, err := jsonwrapper.NewObjectFromBytes(body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"failure":{"msg":"Bad request"}}`)
			return
		}
		m_type_number, err := json.GetInt64("type")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"failure":{"msg":"Bad request"}}`)
			return
		}
		m_type := int(m_type_number)
		endpoint, err := json.GetString("endpoint")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"failure":{"msg":"Bad request"}}`)
			return
		}
		msg, err := json.GetString("message")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"failure":{"msg":"Bad request"}}`)
			return
		}
		exp, err := json.GetInt64("expiration")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"failure":{"msg":"Bad request"}}`)
			return
		}

		obj, err := message.NewMessageObject(m_type, endpoint, msg, exp)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"failure":{"msg":"`+err.Error()+`"}}`)
			return
		}

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"failure":{"msg":"Bad request"}}`)
			return
		}

		log.Println("distributing...")

		_, err = clustering.DistCalls(obj)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"failure":{"msg":"`+err.Error()+`"}}`)
			return
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"success":{"msg": "Message to `+obj.Endpoint+
				` scheduled successfully"}}`)
			return
		}
	}
}

func routes() {
	http.HandleFunc("/", handler)
	//http.HandleFunc("/ws", handlerWs)
}
