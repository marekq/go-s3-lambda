package main

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
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
	"github.com/aws/aws-xray-sdk-go/xray"
	"golang.org/x/net/context/ctxhttp"
)

var (
	// retrieve the ddb table name
	ddbtable = os.Getenv("ddbtable")
)

// main handler
func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	os.Setenv("AWS_XRAY_CONTEXT_MISSING", "LOG_ERROR")
	_, Seg1 := xray.BeginSubsegment(ctx, "main")

	type Item struct {
		Fileurl  string
		Filesize int
		Md5      string
	}

	// if no sqs messages are submitted, exit
	if len(sqsEvent.Records) == 0 {
		return errors.New("No SQS message passed to function")
	}

	// setup a session with ddb and instrument it with xray
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svcddb := dynamodb.New(sess)
	xray.AWS(svcddb.Client)

	// create a waitgroup
	var wg sync.WaitGroup

	// start processing for every received sqs message
	for _, msg := range sqsEvent.Records {
		wg.Add(1)

		// decode base64 message to a string
		s3urldec, err := base64.StdEncoding.DecodeString(msg.Body)

		// print error if base64 could not be decoded
		if err != nil {
			log.Println("error decoding base64 message")
			log.Println(err)

		} else {

			// retrieve the file over http in a go routine
			_, Seg2 := xray.BeginSubsegment(ctx, "http-get")

			go func() {

				// retrieve the file
				resp, err := ctxhttp.Get(ctx, xray.Client(nil), string(s3urldec))
				if err != nil {
					log.Printf("error %s\n", err)
				}

				filesizeint := resp.ContentLength
				filesizestr := strconv.FormatInt(filesizeint, 10)

				// calculate md5 hash of file
				h := md5.New()
				_, err = io.Copy(h, resp.Body)
				if err != nil {
					log.Printf("md5 error %v", err)
				}
				md5hash := hex.EncodeToString(h.Sum(nil))

				resp.Body.Close()

				// add metadata to xray
				xray.AddMetadata(ctx, "FileURL", string(s3urldec))
				xray.AddMetadata(ctx, "MD5", md5hash)

				// create ddb item struct
				item := Item{
					Fileurl:  string(s3urldec),
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

					log.Println("done - " + string(s3urldec) + " " + md5hash + " " + filesizestr)
				}

				// complete the task
				wg.Done()

			}()

			// close xray subsegment
			Seg2.Close(nil)

		}
		wg.Wait()

	}

	// close xray subsegment
	Seg1.Close(nil)
	return nil
}

// start the lambda handler
func main() {
	lambda.Start(handler)
}
