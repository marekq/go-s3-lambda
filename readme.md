S3 Lambda read benchmark
========================

You can use the two solutions in the repositoru to test S3 read performance from Lambda functions at scale. The difference between both solutins is that one will retrieve the object based on S3 URL using a HTTP GET while the other will use a signed S3 URI using the AWS Go SDK. 

*1. Retrieve based on signed S3 URL*

You can run the generator on your machine to send signed S3 URL's to an SQS queue. The S3 URL's are created by the generator by selecting a random file in the bucket. 

*2. Retrieve based on S3 URI*

You can run the generator on your machine to send S3 URI's to an SQS queue. The S3 URI's are created by the generator by selecting a random file in the bucket. 

Once the messages arrive on the SQS queue, they are processed one by one by the Lambda function. The function will download the contents to memory and calculate a CRC32 hash, which you can retrieve from the CloudWatch logs. 