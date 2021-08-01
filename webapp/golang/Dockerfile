FROM golang:1.16

RUN mkdir -p /home/webapp
COPY . /home/webapp
WORKDIR /home/webapp
RUN make
CMD ./app
