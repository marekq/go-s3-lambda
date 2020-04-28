package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-xray-sdk-go/xray"
	"golang.org/x/net/context/ctxhttp"
)

// main handler
func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	os.Setenv("AWS_XRAY_CONTEXT_MISSING", "LOG_ERROR")
	_, Seg1 := xray.BeginSubsegment(ctx, "main")

	if len(sqsEvent.Records) == 0 {
		return errors.New("No SQS message passed to function")
	}

	var wg sync.WaitGroup

	for _, msg := range sqsEvent.Records {
		wg.Add(1)

		// decode base64 message to a string
		s3urldec, err := base64.StdEncoding.DecodeString(msg.Body)

		if err != nil {
			log.Println("error decoding base64 message")
			log.Println(err)

		} else {

			log.Printf("Got SQS message", string(s3urldec))
			_, Seg2 := xray.BeginSubsegment(ctx, "http-get")

			go func() {
				resp, err := ctxhttp.Get(ctx, xray.Client(nil), string(s3urldec))
				if err != nil {
					log.Printf("error", err)
				}

				crc := crc32.NewIEEE()
				io.Copy(crc, resp.Body)
				hash := crc.Sum32()

				fmt.Println(fmt.Sprint(hash) + " " + string(s3urldec))
				xray.AddMetadata(ctx, "CRC32", hash)
				xray.AddMetadata(ctx, "FileURL", string(s3urldec))

				resp.Body.Close()
				Seg2.Close(nil)
				wg.Done()

			}()
		}
		wg.Wait()

	}

	Seg1.Close(nil)
	return nil
}

func msgshttp(ctx context.Context, s3uri string) {

}

// start the lambda handler
func main() {
	lambda.Start(handler)
}
