#!/bin/bash

USERNAME="sanand24"
HOST=$2
SCRIPT="cd /srv/key-value-store-raft/server/; cat ${1}-pid.txt | xargs kill; rm $1-pid.txt"
ssh -l ${USERNAME} ${HOST} ${SCRIPT}