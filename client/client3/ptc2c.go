package main

import (
	"fmt"
	"strconv"
)

func ptc2c() {
	serverList := []string{
		"localhost:8001",
		"localhost:8002",
		"localhost:8003",
	}
	fmt.Println("Calling init -- ", kv739_init(serverList, 3))
	// var start_time = time.Now()
	var oldValue string

	for i := 2000; i < 3000; i++ {
		var key = strconv.Itoa(i)
		kv739_put(key, key, &oldValue)
		fmt.Println(key)
	}

	var not_found = 0
	var wrong_values_count = 0

	for i := 666; i < 999; i++ {
		var key = strconv.Itoa(i)
		var x = kv739_get(key, &oldValue)
		if x == -1 {
			not_found++
		}

		if key != oldValue {
			fmt.Println("wrong value\n", key, oldValue)
			wrong_values_count++
		}
	}
	fmt.Println("Keys Not Found => ", not_found)
	fmt.Println("Value wrong Found =>", wrong_values_count)
	fmt.Println("done ptc2c")
}
