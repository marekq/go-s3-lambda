package main

import (
	"context"
	"encoding/base64"
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

	// create counters
	sentcount := 0
	bucketcount := 0

	// iterate over the s3 bucket content
	err := s3svc.ListObjectsPages(&s3.ListObjectsInput{Bucket: aws.String(bucket)},
		func(p *s3.ListObjectsOutput, last bool) (shouldContinue bool) {

			for _, item := range p.Contents {

				// create variables from s3 metadata, increase bucketcount by one
				s3uri := *item.Key
				s3size := *item.Size
				s3message := ""
				bucketcount++

				// check if the object on s3 is bigger than 0 bytes
				if s3size != 0 {

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
						// if s3path mode is selected, submit the s3 path to the function
						s3message = s3uri

					} else {

						// no valid lambda mode was found, quitting
						log.Println("invalid lambdamode specified, quitting!")
						os.Exit(1)
					}

					// generate a unique message uuid and send the encoded url to the sqs queue
					_, err := sqssvc.SendMessage(&sqs.SendMessageInput{MessageBody: aws.String(s3message), QueueUrl: aws.String(sqsqueue)})

					// return an error if the message was not sent to sqs
					if err == nil {

						// increase counter by 1 and print message
						sentcount++
						log.Println(strconv.Itoa(sentcount), s3uri, strconv.FormatInt(s3size, 10))

					} else {

						// print error if message failed to send
						log.Println("Failed to send message ", err)

					}

				} else {

					// if the filesize was 0 bytes, skip further processing
					log.Println("Object " + s3uri + " has a filesize of 0 bytes, skipping...")

				}
			}

			return true
		})

	// if s3 listing encountered an error, print it
	if err != nil {
		log.Println(err)
	}

	// print amount of objects found in bucket
	log.Println("found " + strconv.Itoa(bucketcount) + " items in bucket " + bucket)

	// print total sent messages
	log.Println("finished - sent " + strconv.Itoa(sentcount) + " messages")

}

// start the lambda handler
func main() {
	lambda.Start(handler)
}
