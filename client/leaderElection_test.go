package main

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestLeaderElection(t *testing.T) {
	i := 0
	sleepTime := 500 * time.Millisecond
	startTime := time.Now()
	state := &RaftServerSnapshot{}
	var slept *time.Duration

	ServerSetup(2)

	// let servers start up.
	time.Sleep(500 * time.Millisecond)

	for ; state.LeaderIndex < 1 && i < 10; i++ {
		// put server 0 to sleep, let peers react to loss of leader.
		err = ServerSleep(0, sleepTime, slept)

		if err == nil {
			// get current node state from server 1 or 2
			err = ServerCall("GetState", rand.Int()%2+1, 0, state)

			if err == nil {
				if state.LeaderIndex == 0 {
					fmt.Printf("After %s sleep, %d thinks 0 is the leader!\n", sleepTime, state.ServerIndex)
					fmt.Println("state:", state)
					//t.Errorf("After %s sleep, %d thinks 0 is the leader!\n", sleepTime, state.ServerIndex)
				} else if state.LeaderIndex == -1 {
					fmt.Printf("After %s sleep, %d thinks no server is the leader!\n", sleepTime, state.ServerIndex)
					fmt.Println("state:", state)
					//t.Errorf("After %s sleep, %d thinks no server is the leader!", sleepTime, state.ServerIndex)
				}
			} else {
				fmt.Println("Could not get server state:", err)
				// t.Errorf("Could not get server state: %s", err)
			}
		} else {
			fmt.Println("Slept for", slept, "ms", err)
			// t.Errorf("Slept for %d ms: %s", slept, err)
		}
	}

	if i >= 10 {
		t.Errorf("Could not elect a new leader after %d - %s rounds in %s.\n", i, sleepTime, time.Since(startTime))
	} else {
		fmt.Printf("Elected %d as new leader after %d - %s rounds in %s.\n", state.LeaderIndex, i, sleepTime, time.Since(startTime))

	}
	ServerTeardown()
}
