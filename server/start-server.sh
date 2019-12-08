#!/bin/bash

go run server.go sharedTypes.go $1 > $1serverlogs.txt 2>&1 & 
popd