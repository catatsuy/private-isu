FROM golang:1.20

RUN mkdir -p /home/webapp
WORKDIR /home/webapp
COPY go.mod /home/webapp
COPY go.sum /home/webapp
RUN go mod download
COPY . /home/webapp
RUN go install github.com/cosmtrek/air@v1.27.3
CMD ["air", "-c", ".air.toml"]
