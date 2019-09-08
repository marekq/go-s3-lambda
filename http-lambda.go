package main

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

const (
	region = "eu-west-1"
)

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	if len(sqsEvent.Records) == 0 {
		return errors.New("No SQS message passed to function")
	}

	for _, msg := range sqsEvent.Records {
		log.Printf("Got SQS message %q with body %q\n", msg.MessageId, msg.Body)
		msgshttp(msg.Body)
	}

	return nil
}

func msgshttp(s3uri string) {
	resp, err := http.Get(s3uri)

	if err != nil {
		log.Printf("error")
	}

	defer resp.Body.Close()

	crc := crc32.NewIEEE()
	io.Copy(crc, resp.Body)

	//log.Printf("file "+s3uri+" CRC %d\n", crc.Sum32())
	fmt.Println(fmt.Sprint(crc.Sum32()) + "     " + s3uri)
}

func main() {
	lambda.Start(handler)
}
