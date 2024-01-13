FROM golang:latest

ADD entrypoint.sh /entrypoint.sh
ADD bin /lipstick

RUN chmod +x /lipstick/*
RUN chmod +x /entrypoint.sh

