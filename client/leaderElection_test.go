package main

import (
	"fmt"
	"testing"
	"time"
)

func TestLeaderElection(t *testing.T) {
	var slept int
	currState := &RaftServer{}

	ServerSetup()

	time.Sleep(500 * time.Millisecond)
	slept, err = ServerSleep(0, 500)
	if err == nil {
		time.Sleep(500 * time.Millisecond)
		err = GetState(1, currState)

		if err == nil && currState.leaderIndex == 0 {
			fmt.Println(currState)
			t.Errorf("After sleeping for 500 ms, server 0 is still unexpectedly the leader!")
		}
	} else {
		t.Errorf("Slept for %d ms, with %s error", slept, err)
	}

	ServerTeardown()
}
