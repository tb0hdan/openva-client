#!/bin/bash

cd $(dirname $0)

monit -c ./etc/monitrc

while :; do
    ./openva-client -debug
    sleep 3
done
