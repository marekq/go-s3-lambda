package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-xray-sdk-go/xray"
)

var (
	// retrieve the s3 bucket name
	bucket = os.Getenv("s3bucket")
)

// main handler
func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	// enable context missing logging for xray
	os.Setenv("AWS_XRAY_CONTEXT_MISSING", "LOG_ERROR")

	// capture the xray subsegment
	_, Seg1 := xray.BeginSubsegment(ctx, "getMsg")

	// if no sqs records were received as input, print a message and skip processing
	if len(sqsEvent.Records) == 0 {
		log.Println("No SQS message passed to function")

	} else {

		// setup a session with s3 and instrument it with xray
		sess, _ := session.NewSessionWithOptions(session.Options{})

		svc := s3.New(sess)
		xray.AWS(svc.Client)

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
					resp, err := svc.GetObjectWithContext(ctx, &s3.GetObjectInput{
						Bucket: aws.String(bucket),
						Key:    aws.String(s3uri),
					})

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
							log.Printf("md5.go hash.FileMd5 io copy error %v", err)
						}
						md5hash := hex.EncodeToString(h.Sum(nil))

						// print the file and hash output
						log.Println("file", string(s3uri))
						log.Println("md5", md5hash)

						// add metadata to xray
						xray.AddMetadata(ctx, "FileURL", string(s3uri))
						xray.AddMetadata(ctx, "MD5", md5hash)

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
