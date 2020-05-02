package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
	uuid "github.com/satori/go.uuid"
)

var (
	// queue url
	sqsqueue = os.Getenv("sqsqueue")

	// bucket name
	bucket = os.Getenv("s3bucket")

	// get aws region
	region = os.Getenv("AWS_REGION")

	// get lambda mode
	lambdamode = os.Getenv("lambdamode")
)

func handler(ctx context.Context) {
	log.Println("sending to queue " + sqsqueue)

	// create a session with sqs
	sqssess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	sqssvc := sqs.New(sqssess)

	// create a session with s3
	s3sess, _ := session.NewSession(&aws.Config{Region: aws.String(region)})

	s3svc := s3.New(s3sess)

	// get a list of items in the s3 bucket
	resp, err := s3svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(bucket)})
	if err != nil {
		log.Println("Unable to list items in bucket " + bucket)
	}

	// create a counter
	count := 0

	// iterate over the s3 bucket content
	for _, item := range resp.Contents {

		s3uri := *item.Key
		s3size := *item.Size

		// check if the object on s3 is bigger than 0 bytes
		if s3size != 0 {

			s3message := ""

			// if mode is s3signed, create a s3 signed url
			if lambdamode == "s3signed" {
				// create a signed s3 url for the object
				req, _ := s3svc.GetObjectRequest(&s3.GetObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(s3uri),
				})

				// create a signed s3 url for the object with a 60 minute expiration time
				s3sign, err := req.Presign(60 * time.Minute)

				// print an error of the s3 signing failed
				if err != nil {
					log.Println("Failed to sign request ", err)

				} else {

					// encode s3 signed url as base64 string
					s3message = base64.StdEncoding.EncodeToString([]byte(s3sign))
				}

			} else if lambdamode == "s3path" {
				s3message = s3uri

			} else {
				log.Println("invalid lambdamode specified, quitting!")
				os.Exit(1)
			}

			// generate a unique message uuid and send the encoded url to the sqs queue
			uuid1 := fmt.Sprint(uuid.NewV4())
			_, err := sqssvc.SendMessage(&sqs.SendMessageInput{MessageGroupId: aws.String(bucket), MessageDeduplicationId: aws.String(uuid1), MessageBody: aws.String(s3message), QueueUrl: aws.String(sqsqueue)})

			// return an error if the message was not sent to sqs
			if err == nil {

				// increase counter by 1 and print message
				count++
				log.Println(strconv.Itoa(count), s3uri, strconv.FormatInt(s3size, 10))

			} else {

				// print error if message failed to send
				log.Println("Failed to send message ", err)

			}

		} else {

			// if the filesize was 0 bytes, skip further processing
			log.Println("Object " + s3uri + " has a filesize of 0 bytes, skipping...")

		}
	}

	// print amount of objects found in bucket
	log.Println("found ", len(resp.Contents), " items in bucket ", bucket)

	// print total sent messages
	log.Println("finished - sent " + strconv.Itoa(count) + " messages")

}

func main() {
	lambda.Start(handler)
}
