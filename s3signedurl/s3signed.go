package main

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-xray-sdk-go/xray"
	"golang.org/x/net/context/ctxhttp"
)

// main handler
func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	os.Setenv("AWS_XRAY_CONTEXT_MISSING", "LOG_ERROR")
	_, Seg := xray.BeginSubsegment(ctx, "sqs")

	if len(sqsEvent.Records) == 0 {
		return errors.New("No SQS message passed to function")
	}

	for _, msg := range sqsEvent.Records {
		log.Printf("Got SQS message %q with body %q\n", msg.MessageId, msg.Body)
		msgshttp(ctx, msg.Body)
	}

	Seg.Close(nil)
	return nil
}

func msgshttp(ctx context.Context, s3uri string) {
	resp, err := ctxhttp.Get(ctx, xray.Client(nil), s3uri)
	if err != nil {
		log.Printf("error")
	}

	defer resp.Body.Close()

	xray.Capture(ctx, "SendMsg", func(ctx1 context.Context) error {

		crc := crc32.NewIEEE()
		io.Copy(crc, resp.Body)
		hash := crc.Sum32()

		fmt.Println(fmt.Sprint(hash) + " " + s3uri)
		xray.AddMetadata(ctx, "CRC32", hash)
		xray.AddMetadata(ctx, "Filepath", s3uri)
		return nil

	})
}

// start the lambda handler
func main() {
	lambda.Start(handler)
}
