FROM golang:1.19

RUN mkdir -p /home/webapp
COPY go.mod /home/webapp
COPY go.sum /home/webapp
WORKDIR /home/webapp
COPY . /home/webapp
RUN go build -o app
CMD ./app
