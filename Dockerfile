FROM ubuntu:latest

ADD bin/lipstick /bin
ADD bin/lipstickd /bin

ENTRYPOINT [ "/bin/lipstickd" ]
