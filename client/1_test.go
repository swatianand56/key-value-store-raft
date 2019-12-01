package main

import (
	"fmt"
	"strconv"
	"testing"
)

func TestGetKey(t *testing.T) {
	serverList := []string{
		"localhost:8001",
		"localhost:8002",
		"localhost:8003",
	}
	var number_of_keys = 10000

	fmt.Println("Calling init -- ", kv739_init(serverList, 3))
	var oldValue string

	for i := 0; i < number_of_keys; i++ {
		var key = strconv.Itoa(i)
		kv739_put(key, key, &oldValue)
		kv739_get(key, &oldValue)
		fmt.Println(key, oldValue)
		if oldValue != key {
			t.Errorf("Keys not getting inserted or retrieved correctly")
		}
	}
}
