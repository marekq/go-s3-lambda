package main

import (
	"context"
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

func handler(ctx context.Context) {
	log.Println("sending to queue " + sqsqueue)

	// create a session with sqs
	sqssess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	sqssvc := sqs.New(sqssess)

	// create a session with s3
	s3sess, _ := session.NewSession(&aws.Config{})

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

				// if s3 signing was successful, send the message to the sqs queue
				log.Println(s3sign)
				_, err = sqssvc.SendMessage(&sqs.SendMessageInput{MessageBody: aws.String(s3sign), QueueUrl: aws.String(sqsqueue)})

				// return whether the message was sent to sqs
				if err != nil {
					log.Println("Failed to send message ", err)

				} else {

					// increase counter by 1
					count++
					log.Println(strconv.Itoa(count) + " - " + s3uri + " - " + strconv.FormatInt(s3size, 10))
				}
			}

		} else {

			// if the filesize was 0 bytes, skip further processing
			log.Println("Object " + s3uri + " has a filesize of 0 bytes")

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
