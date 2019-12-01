package main

import (
	"fmt"
	"strconv"
)

func ptc2a() {
	serverList := []string{
		"localhost:8001",
		"localhost:8002",
		"localhost:8003",
	}
	fmt.Println("Calling init -- ", kv739_init(serverList, 3))
	// var start_time = time.Now()
	var oldValue string

	for i := 0; i < 333; i++ {
		var key = strconv.Itoa(i)
		kv739_put(key, key, &oldValue)
		fmt.Println(key)
	}
	fmt.Println("done ptc2a")
}
