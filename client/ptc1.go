package main

import (
	"fmt"
	"strconv"
)

func ptc1() {
	fmt.Println("calling ptc1")
	serverList := []string{
		"localhost:8001",
		"localhost:8002",
		"localhost:8003",
	}
	var number_of_keys = 100

	fmt.Println("Calling init -- ", kv739_init(serverList, 3))
	// var start_time = time.Now()
	var oldValue string

	for i := 0; i < number_of_keys; i++ {
		var key = strconv.Itoa(i)
		kv739_put(key, key, &oldValue)
		fmt.Println(key)
	}
	// var end_time = time.Now()

	// var not_found = 0
	// var wrong_values_count = 0

	// for i := 0; i < number_of_keys; i++ {
	// 	var key = strconv.Itoa(i)
	// 	var x = kv739_get(key, &oldValue)
	// 	if x == -1 {
	// 		not_found++
	// 	}

	// 	if key != oldValue {
	// 		fmt.Println("wrong value\n", key, oldValue)
	// 		wrong_values_count++
	// 	}
	// }
	// fmt.Println("Keys Not Found => ", not_found)
	// fmt.Println("Value wrong Found =>", wrong_values_count)

	// var time_elapsed = int(end_time.Sub(start_time))
	// var throughput = number_of_keys / time_elapsed
	// var latency = time_elapsed / number_of_keys

	// fmt.Println("start time, end time, timediff is :", start_time, end_time, time_elapsed)
	// fmt.Println("throughput, latency is : ", throughput, latency)
}
