FROM ubuntu:18.04

RUN apt-get update
RUN apt-get install golang -y
RUN apt-get install sqlite3 -y
RUN apt-get install git -y
RUN mkdir -p /etc/workspace/go
ENV GOPATH=/etc/workspace/go
RUN go get -u -v github.com/brharrelldev/weatherapi
