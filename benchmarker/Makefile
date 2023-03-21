all: bin/benchmarker

bin/benchmarker: *.go checker/*.go
	go build -o bin/benchmarker

bench:
	./bin/benchmarker -u ./userdata -t http://localhost
