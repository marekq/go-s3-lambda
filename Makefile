.PHONY: deps
deps:
	go get -d -v ./s3lambda/
	go get -d -v ./s3path-loadgen/
	go get -d -v ./s3signedurl-loadgen/

.PHONY: clean
clean: 
	rm -rf ./s3lambda/s3lambda
	rm -rf ./s3path-loadgen/s3path-loadgen
	rm -rf ./s3signedurl-loadgen/s3signedurl-loadgen
	
.PHONY: build
build:
	GOOS=linux GOARCH=amd64 go build -o ./s3lambda/s3lambda ./s3lambda
	GOOS=linux GOARCH=amd64 go build -o ./s3path-loadgen/s3path-loadgen ./s3path-loadgen
	GOOS=linux GOARCH=amd64 go build -o s3signedurl-loadgen/s3signedurl-loadgen ./s3signedurl-loadgen

	upx -1 ./s3lambda/s3lambda
	upx -1 ./s3path-loadgen/s3path-loadgen
	upx -1 ./s3signedurl-loadgen/s3signedurl-loadgen
