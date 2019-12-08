#!/bin/bash

USERNAME="sanand24"
HOST=$2
SCRIPT="cd /srv/key-value-store-raft/server/; go run server.go sharedTypes.go ${1} > ${1}serverlogs.txt 2>&1 &"
ssh -l ${USERNAME} ${HOST} ${SCRIPT}