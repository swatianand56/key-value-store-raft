#!/bin/bash

pushd key-value-store-raft
pushd server
go run server.go sharedTypes.go $1 > $1serverlogs.txt 2>&1 & 
popd