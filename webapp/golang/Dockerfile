FROM golang:1.16.4

RUN mkdir -p /home/webapp
COPY . /home/webapp
WORKDIR /home/webapp
RUN make
CMD ./app
