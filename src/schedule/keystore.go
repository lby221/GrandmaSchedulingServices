package schedule

import (
	"container/list"
	"errors"
	"log"
	"sync"
	"time"
)

var (
	ErrorInvalidScheduleObj = errors.New("Invalid schedule object")
)

var data_map = make(map[int64]*list.List)
var time_table = make(map[int]int64)

var RWMutex = new(sync.RWMutex)

func put(s *Schedule) error {
	if s == nil {
		return ErrorInvalidScheduleObj
	}
	log.Println("msg consumed")

	var now = time.Now().UnixNano() / 1000000

	if s.Exp > 60*60*24*30*1000 {
		s.Exp = 60 * 60 * 24 * 30 * 1000
	}

	var exp = (s.Exp + now) / 100

	RWMutex.Lock()
	t, ttok := time_table[s.Id]
	_, dmok := data_map[exp]

	if ttok == true {
		temp_l := data_map[t]
		for e := temp_l.Front(); e != nil; e = e.Next() {
			if e.Value == s.Id {
				temp_l.Remove(e)
			}
		}
		if temp_l.Len() == 0 {
			delete(data_map, t)
		}
	}

	if dmok != true {
		data_map[exp] = list.New()
	}

	time_table[s.Id] = exp
	data_map[exp].PushBack(s.Id)

	log.Printf("Key store accessed and created record at %d", exp)
	RWMutex.Unlock()

	return nil
}

func get(id int) *Schedule {
	RWMutex.RLock()
	t, ttok := time_table[id]
	RWMutex.RUnlock()

	if ttok == false {
		return nil
	} else {
		return &Schedule{id, t, make(chan bool, 1)}
	}
}

func getAll() []*Schedule {
	RWMutex.RLock()
	size := len(time_table)

	if size == 0 {
		return nil
	}

	var ret = make([]*Schedule, size, size)
	var count = 0
	for key := range time_table {
		ret[count] = &Schedule{key, time_table[key], make(chan bool, 1)}
		count++
	}

	RWMutex.RUnlock()

	return ret
}
