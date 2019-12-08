package main

import (
	"fmt"
	"os"
	"strconv"
)

var servers = make(map[string]*os.Process)

func ServerSetup(debugFlag int, serverList []string, serverIndexes []string) {
	// see sharedTypes.go::VerboseFlags:
	//
	// ALL          1
	// HEARTBEATS   2
	// CONNECTIONS  4
	// READS        8
	// WRITES       16
	//

	for i := range serverIndexes {
		index := serverIndexes[i]
		attr := new(os.ProcAttr)
		attr.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr}
		server, err := os.StartProcess("../../server/server", []string{"../../server/server", index, strconv.Itoa(debugFlag)}, attr)
		if err != nil {
			fmt.Println("Server Error:", index, err)
		}
		servers[index] = server
	}
}

func ServerTeardown(serverIndexes []string) {
	for i := range serverIndexes {
		index := serverIndexes[i]
		if servers[index] != nil {
			servers[index].Kill()
		}
	}
}
