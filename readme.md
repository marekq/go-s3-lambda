S3 Lambda read benchmark
========================

You can use the two solutions in the repositoru to test S3 read performance from Lambda functions at scale. The difference between both solutins is that one will retrieve the object based on S3 URL using a HTTP GET while the other will use a signed S3 URI using the AWS Go SDK. Whenever you run one of the generator functions, all file paths or signed S3 URL's for files within that bucket are submitted to SQS.

Below are the two options that can be used for processing, which are deployed each with their separate generator and reader functions. In addition, a separate SQS queue is deployed for each variant;


*1. Retrieve based on signed S3 URL*

You can run the generator on your machine to send signed S3 URL's to an SQS queue based on the bucket content. The S3 URL's are created by the generator and sent to the SQS queue.  


*2. Retrieve based on S3 URI*

You can run the generator on your machine to send S3 URI's to an SQS queue based on the bucket content. The S3 URI's are created by the generator and sent to the SQS queue. 


Once the messages arrive on the SQS queue, they are processed one by one by the Lambda function. The function will download the contents to memory and calculate a CRC32 hash, which you can retrieve from the CloudWatch logs. 


Installation
------------

You need to have AWS SAM, Go and 'upx' installed on your local machine. In the future, a Docker based build process will be added to to make the build process easier and more repeatable on different machines. 


Next, run 'make deps', 'make build' and 'sam deploy -g' in your local directory. Afterwards, you can run one of the loadgen Lambda functions to send messages to the SQS queue. Remember to rerun 'make build' whenever you made changes to any of the Go source code to build the functions. 


Roadmap
-------

- [ ] Add a DynamoDB table where the filename, filesize and filehash are added for tracking of this metadata. 
- [ ] Add MD5 fingerprinting of files, which is more practical than the current CRC32 code. 
- [ ] Add sensible limits to the Lambda generator functions, i.e. only submit files that were modified in the last 24 hours. Currently the generator Lambda can submit up to a 1000 files for processing. 
- [X] Add XRay tracing to the processing functions for granular performance tuning.


Contact
-------

In case you have any suggestions, questions or remarks, please raise an issue or reach out to @marekq.

