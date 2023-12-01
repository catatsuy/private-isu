FROM golang:1.21

RUN apt update && \
    apt install -y gv graphviz
RUN mkdir -p /home/webapp
WORKDIR /home/webapp
COPY . /home/webapp
RUN go build -o app
CMD ./app
