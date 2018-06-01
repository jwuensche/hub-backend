FROM golang:latest

WORKDIR /go/src/hub-go
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["hub-go"]
