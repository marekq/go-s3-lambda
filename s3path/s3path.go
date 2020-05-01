package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-xray-sdk-go/xray"
)

var (
	// retrieve the s3 bucket name
	bucket   = os.Getenv("s3bucket")
	ddbtable = os.Getenv("ddbtable")
)

// main handler
func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	// enable context missing logging for xray
	os.Setenv("AWS_XRAY_CONTEXT_MISSING", "LOG_ERROR")

	type Item struct {
		Fileurl  string
		Filesize int
		Md5      string
	}

	// capture the xray subsegment
	_, Seg1 := xray.BeginSubsegment(ctx, "getMsg")

	// if no sqs records were received as input, print a message and skip processing
	if len(sqsEvent.Records) == 0 {
		log.Println("No SQS message passed to function")

	} else {

		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))

		// setup a session with s3 and instrument it with xray
		svcs3 := s3.New(sess)
		xray.AWS(svcs3.Client)

		// setup a session with ddb and instrument it with xray
		svcddb := dynamodb.New(sess)
		xray.AWS(svcddb.Client)

		var wg sync.WaitGroup

		// print the md5 hash of every file
		for _, msg := range sqsEvent.Records {
			wg.Add(1)

			// retrive the s3uri from the sqs message
			s3uri := msg.Body

			// run a go routine to retrieve the s3 file and calculate the hash
			go func() {

				// capture the s3 get and md5 calculation with xray
				xray.Capture(ctx, "SendMsg", func(ctx1 context.Context) error {
					resp, err := svcs3.GetObjectWithContext(ctx, &s3.GetObjectInput{
						Bucket: aws.String(bucket),
						Key:    aws.String(s3uri),
					})

					filesizeint := *resp.ContentLength
					filesizestr := strconv.FormatInt(filesizeint, 10)

					// print error if the file could not be retrieved
					if err != nil {
						log.Printf("error %s\n", err)

					} else {

						// instrument the md5 calculation with xray
						_, Seg2 := xray.BeginSubsegment(ctx, "MD5")

						// calculate the md5 hash
						h := md5.New()
						_, err = io.Copy(h, resp.Body)
						if err != nil {
							log.Printf("md5 error %v", err)
						}
						md5hash := hex.EncodeToString(h.Sum(nil))

						// print the file and hash output
						//log.Println("file", string(s3uri))
						//log.Println("md5", md5hash)
						//log.Println("filesize", filesizestr)

						// add metadata to xray
						xray.AddMetadata(ctx, "fileurl", string(s3uri))
						xray.AddMetadata(ctx, "md5", md5hash)
						xray.AddMetadata(ctx, "filesize", filesizestr)

						// create ddb item struct
						item := Item{
							Fileurl:  s3uri,
							Md5:      md5hash,
							Filesize: int(filesizeint),
						}

						// marshal the ddb items
						av, err := dynamodbattribute.MarshalMap(item)

						if err != nil {
							log.Println(err)
						}

						// put the item into dynamodb
						_, err = svcddb.PutItemWithContext(ctx, &dynamodb.PutItemInput{
							Item:      av,
							TableName: aws.String(ddbtable),
						})

						// print the success or error message from the put to ddb
						if err != nil {

							log.Println("Got error calling PutItem:")
							log.Println(err.Error())
						} else {

							log.Println("done - " + s3uri + " " + md5hash + " " + filesizestr)
						}

						// close the xray subsegment
						Seg2.Close(nil)
					}

					return nil

				})

				// set the task as done
				wg.Done()

			}()
		}

		// wait for waitgroups to complete
		wg.Wait()

	}

	// close the xray subsegment
	Seg1.Close(nil)

	return nil
}

// start the lambda handler
func main() {
	lambda.Start(handler)
}
