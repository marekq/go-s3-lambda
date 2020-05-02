package main

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-xray-sdk-go/xray"
	"golang.org/x/net/context/ctxhttp"
)

var (
	// retrieve the s3 bucket name
	bucket = os.Getenv("s3bucket")

	// retrieve the ddb table name
	ddbtable = os.Getenv("ddbtable")

	// retrieve the lambda mode (s3 signed urls or s3 uri)
	lambdamode = os.Getenv("lambdamode")
)

// main handler
func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	// get the unix ts for start time
	startts := int32(time.Now().UnixNano() / 1000000)

	// set a variable for total file size
	filesz := 0

	// set a variable for file count
	count := 0

	// enable context missing logging for xray
	os.Setenv("AWS_XRAY_CONTEXT_MISSING", "LOG_ERROR")

	type Item struct {
		Fileurl  string
		Filesize int
		Md5      string
		Bucket   string
	}

	// capture the xray subsegment
	_, Seg1 := xray.BeginSubsegment(ctx, "handler")

	// if no sqs records were received as input, print a message and skip processing
	if len(sqsEvent.Records) == 0 {
		log.Println("No SQS message passed to function")

		// if sqs messages were received, continue
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
				xray.Capture(ctx, "GetMsg", func(ctx1 context.Context) error {

					// create empty values
					filesizestr := ""
					filesizeint := int64(0)
					md5hash := ""

					// if the sqs queue contains s3 paths, retrieve over s3
					if lambdamode == "s3path" {
						resp, err := svcs3.GetObjectWithContext(ctx1, &s3.GetObjectInput{
							Bucket: aws.String(bucket),
							Key:    aws.String(s3uri),
						})

						filesizeint = *resp.ContentLength
						filesizestr = strconv.FormatInt(filesizeint, 10)

						// add object filesize to total counter
						filesz += int(filesizeint)

						// instrument the md5 calculation with xray
						_, Seg2 := xray.BeginSubsegment(ctx1, "calc-md5")

						// calculate the md5 hash
						h := md5.New()
						_, err = io.Copy(h, resp.Body)

						if err != nil {
							log.Printf("md5 error %v", err)
						}

						md5hash = hex.EncodeToString(h.Sum(nil))
						Seg2.Close(nil)

					} else if lambdamode == "s3signed" {
						// if the sqs queue contains s3 signed urls

						// decode base64 message to a string
						s3urldec, _ := base64.StdEncoding.DecodeString(s3uri)

						// get the s3 filename from the s3 signed url
						s3tmp := strings.Split(string(s3urldec), "?")[0]
						s3file := strings.Trim(fmt.Sprint((strings.Split(s3tmp, "/")[4:])), "[]")
						s3uri = s3file

						// retrieve the file
						resp, err := ctxhttp.Get(ctx1, xray.Client(nil), string(s3urldec))
						if err != nil {
							log.Printf("error %s\n", err)
						}

						filesizeint := resp.ContentLength
						filesizestr = strconv.FormatInt(filesizeint, 10)

						// add object filesize to total counter
						filesz += int(filesizeint)

						// instrument the md5 calculation with xray
						_, Seg3 := xray.BeginSubsegment(ctx1, "calc-md5")

						// calculate the md5 hash
						h := md5.New()
						_, err = io.Copy(h, resp.Body)

						if err != nil {
							log.Printf("md5 error %v", err)
						}

						md5hash = hex.EncodeToString(h.Sum(nil))
						Seg3.Close(nil)

					} else {
						log.Println("invalid lambdamode specified, quitting!")
						os.Exit(1)
					}

					// add metadata to xray
					xray.AddMetadata(ctx1, "fileurl", string(s3uri))
					xray.AddMetadata(ctx1, "md5", md5hash)
					xray.AddMetadata(ctx1, "filesize", filesizestr)

					// dynamodb put segment
					_, Seg4 := xray.BeginSubsegment(ctx1, "dynamodb-put")

					// create ddb item struct
					item := Item{
						Fileurl:  s3uri,
						Md5:      md5hash,
						Filesize: int(filesizeint),
						Bucket:   bucket,
					}

					// marshal the ddb items
					av, err := dynamodbattribute.MarshalMap(item)

					if err != nil {
						log.Println(err)
					}

					// put the item into dynamodb
					_, err = svcddb.PutItemWithContext(ctx1, &dynamodb.PutItemInput{
						Item:      av,
						TableName: aws.String(ddbtable),
					})

					Seg4.Close(nil)

					// print the success or error message from the put to ddb
					if err != nil {

						log.Println("got error calling dynamodb:putitem ")
						log.Println(err.Error())

					} else {

						count++
						log.Println("done - " + s3uri + " " + md5hash + " " + filesizestr)
					}

					// complete the task
					wg.Done()

					return nil

				})

			}()
		}

		// wait for waitgroups to complete
		wg.Wait()

	}

	// get the unix ts for end time and total time
	endts := int32(time.Now().UnixNano() / 1000000)
	totaltime := fmt.Sprint(endts - startts)

	log.Println("exit - processed " + strconv.Itoa(count) + " messages with " + fmt.Sprint(filesz/1024) + " KB in " + totaltime + " msec")
	xray.AddMetadata(ctx, "filecount", count)
	xray.AddMetadata(ctx, "totalfilesize", filesz)

	Seg1.Close(nil)
	return nil
}

// start the lambda handler
func main() {
	lambda.Start(handler)
}
