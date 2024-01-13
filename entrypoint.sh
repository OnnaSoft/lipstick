#!/bin/sh


if [ "$1" = "server" ]; then
    /lipstick/lipstick-server
elif [ "$1" = "client" ]; then
    /lipstick/lipstick-client
else
  exit 1
fi

