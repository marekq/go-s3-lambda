.PHONY: deps
deps:
	go get -d -v ./s3path/
	go get -d -v ./s3path-loadgen/
	go get -d -v ./s3signedurl/
	go get -d -v ./s3signedurl-loadgen/

.PHONY: clean
clean: 
	rm -rf ./s3path/s3path
	rm -rf ./s3path-loadgen/s3path-loadgen
	rm -rf ./s3signedurl/s3signedurl
	rm -rf ./s3signedurl-loadgen/s3signedurl-loadgen
	
.PHONY: build
build:
	#s3path
	GOOS=linux GOARCH=amd64 go build -o ./s3path/s3path ./s3path
	GOOS=linux GOARCH=amd64 go build -o ./s3path-loadgen/s3path-loadgen ./s3path-loadgen

	#s3signedurl
	GOOS=linux GOARCH=amd64 go build -o s3signedurl/s3signedurl ./s3signedurl
	GOOS=linux GOARCH=amd64 go build -o s3signedurl-loadgen/s3signedurl-loadgen ./s3signedurl-loadgen

	# compress go binaries with upx to reduce filesize
	upx -1 ./s3path/s3path
	upx -1 ./s3path-loadgen/s3path-loadgen

	upx -1 ./s3signedurl/s3signedurl
	upx -1 ./s3signedurl-loadgen/s3signedurl-loadgen
