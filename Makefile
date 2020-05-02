.PHONY: deps
deps:
	go get -d -v ./s3lambda/
	go get -d -v ./s3loadgen/

.PHONY: clean
clean: 
	rm -rf ./s3lambda/s3lambda
	rm -rf ./s3loadgen/s3loadgen
	
.PHONY: build
build:
	GOOS=linux GOARCH=amd64 go build -o ./s3lambda/s3lambda ./s3lambda
	GOOS=linux GOARCH=amd64 go build -o ./s3loadgen/s3loadgen ./s3loadgen

	upx -1 ./s3lambda/s3lambda
	upx -1 ./s3loadgen/s3loadgen
