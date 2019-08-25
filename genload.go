package main

import (
	"log"
	"math/rand"
	"strconv"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

const (
	// queue url
	q = "https://sqs.eu-west-1.amazonaws.com/123456789012/sqs"

	// region
	r = "eu-west-1"

	// bucket name
	b = "<bucket>"

	// bucket prefix
	p = "data"

	// amount of retrieves per go routine
	c = 100

	// total amount of go routines
	d = 100
)

//
// do not change anything below this line
//

func main() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	// create a session with sqs
	svc1 := sqs.New(sess)

	// create a session with s3 (use only if s3 signing is used, commented by default)
	//svc2 := s3.New(sess)

	for a := 0; a < c; a++ {

		// create a wait group to wait for go subroutines
		var wg sync.WaitGroup

		// spawn go routines depending on total count
		for b := 0; b < d; b++ {

			// add one count to the workgroup
			wg.Add(1)

			// run the send message command in parallel
			go func() {
				defer wg.Done()
				ri := strconv.Itoa(rand.Intn(99))

				s3uri := p + "/" + ri + ".file"
				log.Println("The S3 URI is", s3uri)

				// if you want to send a signed s3 url instead of the s3 uri, uncomment the following block
				/*
					req, _ := svc2.GetObjectRequest(&s3.GetObjectInput{
						Bucket: aws.String(b),
						Key:    aws.String(s3uri),
					})
					s3uri, err := req.Presign(15 * time.Minute)

					if err != nil {
						log.Println("Failed to sign request", err)
					}
				*/

				// send the message to the sqs queue
				svc1.SendMessage(&sqs.SendMessageInput{MessageBody: aws.String(s3uri), QueueUrl: aws.String(q)})
			}()

		}
		// wait for all routines to finish
		wg.Wait()

	}
}
