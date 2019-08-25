package main

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"net/url"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	region = "eu-west-1"
)

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	if len(sqsEvent.Records) == 0 {
		return errors.New("No SQS message passed to function")
	}

	for _, msg := range sqsEvent.Records {
		fmt.Printf("Got SQS message %q with body %q\n", msg.MessageId, msg.Body)
		fn := url.QueryEscape(msg.Body)
		log.Printf("Checking " + msg.Body)
		msgss3(msg.Body, fn)
	}

	return nil
}

func msgss3(s3uri string, fn string) {
	bucket := os.Getenv("bucket")

	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{Region: aws.String("eu-west-1")},
	})

	svc := s3.New(sess)

	out, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(s3uri),
	})

	if err != nil {
		log.Printf("error\n")
	}

	crc := crc32.NewIEEE()
	io.Copy(crc, out.Body)

	log.Printf("file "+fn+" CRC %d\n", crc.Sum32())
}

func main() {
	lambda.Start(handler)
}
