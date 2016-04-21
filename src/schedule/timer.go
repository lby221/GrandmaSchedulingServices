package schedule

import (
	"log"
	"time"
)

const breakpoint int = 500

var main_timer = time.NewTicker(time.Millisecond * 60)
var current_time int64
var counter int = 0

func startLoop() {
	go func() {
		for _ = range main_timer.C {
			tickingFunc()
		}
	}()
}

func endLoop() {
	main_timer.Stop()
}

func tickingFunc() {
	if counter == breakpoint {
		counter = 0
		current_time = time.Now().UnixNano()/1000000 - 1000
	} else {
		counter = counter + 1
		current_time = current_time + 62
	}

	l, dmok := data_map[current_time/100]
	if dmok == false {
		return
	} else {
		go func() {
			for e := l.Front(); e != nil; e = e.Next() {
				s := get(e.Value.(int))
				log.Printf("push scheduled msg at %d", current_time/100)
				go s.pushToSendingQueue()
				delete(time_table, s.Id)
			}
			delete(data_map, current_time/100)
		}()
	}
}
