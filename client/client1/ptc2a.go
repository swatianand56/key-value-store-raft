package main

import (
	"fmt"
	"strconv"
	"time"
)

func ptc2a() {
	serverList := []string{
		"10.10.1.1:8001",
		"10.10.1.2:8002",
		"10.10.1.3:8003",
		"10.10.1.1:8004",
		"10.10.1.2:8005",
		"10.10.1.3:8006",
		"10.10.1.1:8007",
		"10.10.1.2:8008",
		"10.10.1.3:8009",
		"10.10.1.1:8010",
		"10.10.1.2:8011",
	}
	fmt.Println("Calling init -- ", kv739_init(serverList, 3))
	// var start_time = time.Now()
	var oldValue string
	// var number_of_keys = 100

	var not_found = 0
	var wrong_values_count = 0
	var put_failures = 0

	start := time.Now()

	for i := 0; i < 10000; i++ {
		var key = strconv.Itoa(i)
		var result = kv739_put(key, key, &oldValue)
		if result == -1 {
			put_failures++
		}
	}

	put_end := time.Now()

	for i := 0; i < 10000; i++ {
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

	fmt.Println("Keys Not Found => ", not_found)
	fmt.Println("Value wrong Found =>", wrong_values_count)
	fmt.Println("Put Key request failures => ", put_failures)
	fmt.Println("Put Elapsed ", put_elapsed, "get elapsed", get_elapsed)
	fmt.Println("done ptc2a")
}
