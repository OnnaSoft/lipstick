FROM ubuntu:24.04

COPY bin/lipstick /bin
COPY bin/lipstickd /bin

ENTRYPOINT [ "/bin/lipstickd" ]
