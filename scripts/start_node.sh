#!/bin/sh

config_dir="/home/coin/haqqd"

if [ ! -d "$config_dir" ]
then
  echo "Node config directory does NOT exist!"
  echo "Running forever (hit Ctrl-C to exit) ..."
  until [ -d "$config_dir" ]
  do
    sleep 5
  done
  echo "Node config directory finally created!"
  sleep 15
fi

haqqd --home "$config_dir" start

