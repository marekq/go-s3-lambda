appname='s3-golang'

GOOS=linux go build main.go 
zip -r9 main.zip main
aws lambda update-function-code --function-name $appname --zip-file fileb://main.zip 
