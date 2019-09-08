
# set the lambda function name
appname='s3-golang-beta-s3'

cp http-lambda.go main.go

# build the go binary for linux that can run on lambda
GOOS=linux go build -ldflags="-s -w" main.go 

# compress the go binary
upx -1 main

# zip the go binary and source code
zip -r9 main.zip main

# update the lambda function
aws lambda update-function-code --function-name $appname --zip-file fileb://main.zip 
