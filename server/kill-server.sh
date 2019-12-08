#!/bin/bash

cat $1-pid.txt | xargs kill
rm $1-pid.txt