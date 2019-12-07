package main

/*
the leader must accept log entries from clients and replicate them across the cluster, forcing the other logs to agree with its own.
*/

import (
	"testing"
	"time"
)

func TestLogReplication(t *testing.T) {
	//var putRequest KeyValuePair
	putRequest := &KeyValuePair{"akey", "avalue"}
	var value *string

	ServerSetup(0)

	// let servers start up.
	time.Sleep(500 * time.Millisecond)

	// put a key on the server.
	err = nil
	err = ServerCall("PutKey", 0, putRequest, value)

	if err == nil {
		time.Sleep(500 * time.Millisecond)

		// get key from the server.
		err = ServerCall("GetKeyFromAny", 1, "akey", value)

		if err == nil {
			if *value != "avalue" {
				t.Errorf("Unexpected value 'avalue' != '%s'\nerr: %s", *value, err)
			}
		} else {
			t.Errorf("Server 1 failed: %s", err)
		}
	} else {
		t.Errorf("Server 0 failed: %s", err)
	}

	ServerTeardown()
}
