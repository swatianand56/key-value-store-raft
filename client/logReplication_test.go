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
	var value = ""
	clusterSize := 3
	flags := 0 // VERBOSE.HEARTBEATS + VERBOSE.LIVENESS + VERBOSE.STATE
	leaderIndex := 0
	debugMessage(flags, -1, VERBOSE.STATE, "Starting log replication test.")

	leader, active, _ := ServerSetup(clusterSize, flags)
	if leader < 0 {
		t.Errorf("Failed test: Couldn't elect a leader in 10 rounds.")
	} else {
		for found := false; leaderIndex < len(active) && found == false; leaderIndex++ {
			found = (active[leaderIndex] == leader)
		}
		leaderIndex--

		// put a key on the server.
		err = ServerCallLong("PutKey", active[leaderIndex], putRequest, &value)

		if err == nil {
			// let the key propagate
			time.Sleep(500 * time.Millisecond)

			// cheat and get key from non-leader server.
			leaderIndex = (leaderIndex + 1) % len(active)
			err = ServerCallLong("GetKeyFromAny", active[leaderIndex], "akey", &value)

			if err == nil {
				if value != "avalue" {
					t.Errorf("Unexpected value 'avalue' != '%s'\nerr: %s", value, err)
				}
			} else {
				t.Errorf("Follower failed: %s", err)
			}
		} else {
			t.Errorf("Leader PutKey failed: %s", err)
		}

		debugMessage(flags, -1, VERBOSE.STATE, "Finished log replication test.")
	}

	ServerTeardown(flags)
}
