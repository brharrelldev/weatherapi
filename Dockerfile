FROM ubuntu:18.04

RUN apt-get update
RUN apt-get install golang -y
RUN apt-get install sqlite3 -y
RUN apt-get install git -y
RUN mkdir -p /etc/workspace/go
ENV GOPATH=/etc/workspace/go
RUN go get -u -v github.com/brharrelldev/weatherapi
RUN cd /etc/workspace/go/src/github.com/brharrelldev/weatherapi/
ENTRYPOINT ["go run /etc/workspace/go/src/github.com/brharrelldev/weatherapi/main.go"]