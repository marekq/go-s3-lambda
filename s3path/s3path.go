package main

import (
	"context"
	"errors"
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

func getmsg(s3uri string) {
	log.Printf("Got SQS message " + s3uri)

	ctx := context.Background()
	// setup a session with s3 and instrument it with xray
	sess, _ := session.NewSessionWithOptions(session.Options{})

	svc := s3.New(sess)
	xray.AWS(svc.Client)

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

			// calculate the crc32 hash
			crc := crc32.NewIEEE()
			io.Copy(crc, out.Body)

			// print the crc32 hash
			log.Printf("file", s3uri, "CRC32 %d\n", crc.Sum32())
			xray.AddMetadata(ctx, "CRC32", crc.Sum32())
			xray.AddMetadata(ctx, "Filepath", s3uri)
		}

		return nil
	})
}

// main handler
func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	// enable context missing logging for xray
	os.Setenv("AWS_XRAY_CONTEXT_MISSING", "LOG_ERROR")

	// capture the xray subsegment
	_, Seg := xray.BeginSubsegment(ctx, "sqs")

	// if no sqs records were received as input, return an error
	if len(sqsEvent.Records) == 0 {
		return errors.New("No SQS message passed to function")
	}

	var wg sync.WaitGroup

	// print the crc32 hash of every file
	for _, msg := range sqsEvent.Records {
		wg.Add(1)

		go getmsg(msg.Body)
		wg.Done()

	}

	wg.Wait()

	Seg.Close(nil)
	return nil
}

func msgss3(ctx context.Context, s3uri string) {

}

// start the lambda handler
func main() {
	lambda.Start(handler)
}
