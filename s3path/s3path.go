package main

import (
	"context"
	"hash/crc32"
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

		// print the crc32 hash of every file
		for _, msg := range sqsEvent.Records {
			wg.Add(1)

			// retrive the s3uri from the sqs message
			s3uri := msg.Body
			log.Printf("Got SQS message " + s3uri)

			// run a go routine to retrieve the s3 file and calculate the hash
			go func() {

				// capture the s3 get and crc32 calculation with xray
				xray.Capture(ctx, "SendMsg", func(ctx1 context.Context) error {
					out, err := svc.GetObjectWithContext(ctx, &s3.GetObjectInput{
						Bucket: aws.String(bucket),
						Key:    aws.String(s3uri),
					})

					// print error if the file could not be retrieved
					if err != nil {
						log.Printf("error ", err)
					} else {

						// instrument the crc32 calculation with xray
						_, Seg2 := xray.BeginSubsegment(ctx, "CRC32")

						// calculate the crc32 hash
						crc := crc32.NewIEEE()
						io.Copy(crc, out.Body)

						// print the crc32 hash
						log.Printf("file", s3uri, "CRC32 %d\n", crc.Sum32())
						xray.AddMetadata(ctx1, "CRC32", crc.Sum32())
						xray.AddMetadata(ctx1, "Filepath", s3uri)

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
