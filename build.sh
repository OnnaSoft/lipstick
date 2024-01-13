#!/bin/sh

rm bin/* -rf

go build -o bin/lipstick-client client/main.go
go build -o bin/lipstick-server server/main.go 

