package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
)

var (
	// queue url
	sqsqueue = os.Getenv("sqsqueue")

	// bucket name
	bucket = os.Getenv("s3bucket")
)

func handler() {
	log.Println("sending to queue " + sqsqueue)

	sqssess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	// create a session with sqs
	sqssvc := sqs.New(sqssess)

	// setup an s3 session
	s3sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("eu-west-1")},
	)

	// create a session with s3
	s3svc := s3.New(s3sess)

	// Get a list of items in the s3 bucket
	resp, err := s3svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(bucket)})
	if err != nil {
		log.Println("Unable to list items in bucket " + bucket)
	}

	// create a counter
	count := 0

	for _, item := range resp.Contents {

		s3uri := *item.Key
		s3size := *item.Size

		// create a signed s3 url for the object
		req, _ := s3svc.GetObjectRequest(&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(s3uri),
		})

		s3sign, err := req.Presign(15 * time.Minute)

		if err != nil {
			log.Println("Failed to sign request ", err)
		}

		// send the message to the sqs queue
		_, err = sqssvc.SendMessage(&sqs.SendMessageInput{MessageBody: aws.String(s3sign), QueueUrl: aws.String(sqsqueue)})

		if err != nil {
			log.Println("Failed to send message ", err)
		} else {
			count++
			log.Println(strconv.Itoa(count) + " - " + s3uri + " - " + strconv.FormatInt(s3size, 10))
		}
	}

	log.Println("found ", len(resp.Contents), " items in bucket ", bucket)

	// print total sent messages
	log.Println("finished - sent " + strconv.Itoa(count) + " messages")

}

func main() {
	lambda.Start(handler)
}
