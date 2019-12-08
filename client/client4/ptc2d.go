package main

import (
	"fmt"
	"strconv"
	"time"
)

func ptc2d() {
	serverList := []string{
		"localhost:8001",
		"localhost:8002",
		"localhost:8003",
	}
	fmt.Println("Calling init -- ", kv739_init(serverList, 3))
	// var start_time = time.Now()
	var oldValue string
	var not_found = 0
	var wrong_values_count = 0
	var put_failures = 0
	// var number_of_keys = 100

	start := time.Now()

	for i := 30000; i < 40000; i++ {
		var key = strconv.Itoa(i)
		var result = kv739_put(key, key, &oldValue)
		if result == -1 {
			put_failures++
		}
	}

	put_end := time.Now()

	for i := 0; i < 333; i++ {
		var key = strconv.Itoa(i)
		var x = kv739_get(key, &oldValue)
		if x == -1 {
			not_found++
		}

		if key != oldValue {
			wrong_values_count++
		}
	}

	get_end := time.Now()
	put_elapsed := put_end.Sub(start)
	get_elapsed := get_end.Sub(put_end)

	fmt.Println("Put Key request failures => ", put_failures)
	fmt.Println("Put Elapsed ", put_elapsed, "get elapsed", get_elapsed)

	fmt.Println("Keys Not Found => ", not_found)
	fmt.Println("Value wrong Found =>", wrong_values_count)

	fmt.Println("done ptc2d")
}
