#!/bin/sh

rm bin/* -rf

go build -o bin/lipstick client/main.go
go build -o bin/lipstickd server/main.go 

chmod +x bin/lipstick
chmod +x bin/lipstickd
