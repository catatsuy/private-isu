FROM golang:1.17

RUN mkdir -p /home/webapp
COPY . /home/webapp
WORKDIR /home/webapp
RUN go build -o app
CMD ./app
