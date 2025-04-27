package storage

import "sync"

var (
	Mutex       sync.Mutex
	SensorData  = make(map[string]map[string]string)
	BroadcastCh = make(chan string, 100)
)
