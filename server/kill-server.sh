#!/bin/bash

pushd key-value-store-raft
pushd server
cat $1-pid.txt | xargs kill
rm $1-pid.txt
popd