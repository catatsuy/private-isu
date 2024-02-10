FROM golang:1.22

RUN mkdir -p /home/webapp
WORKDIR /home/webapp

COPY go.mod /home/webapp/go.mod
COPY go.sum /home/webapp/go.sum
RUN go mod download

COPY . /home/webapp
RUN go build -o app
CMD ./app
