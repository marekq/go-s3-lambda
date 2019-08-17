package main

import (
	"math/rand"
	"strconv"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

const (
	q = "https://sqs.eu-west-1.amazonaws.com/929972393034/wt"
	r = "eu-west-1"
	c = 5000
)

// send a message with a random value
func sendmsg(svc *sqs.SQS) (*sqs.SendMessageOutput, error) {
	ri := strconv.Itoa(rand.Intn(99))

	para := &sqs.SendMessageInput{
		MessageBody: aws.String(ri),
		QueueUrl:    aws.String(q),
	}
	return svc.SendMessage(para)
}

func main() {

	// create a session with sqs
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := sqs.New(sess)

	// create a wait group to wait for go subroutines
	var wg sync.WaitGroup

	// spawn go routines depending on total count
	for i := 0; i < c; i++ {

		// add one count to the workgroup
		wg.Add(1)

		// run the send message command in parallel
		go func() {
			defer wg.Done()
			ri := strconv.Itoa(rand.Intn(99))

			svc.SendMessage(&sqs.SendMessageInput{MessageBody: aws.String(ri), QueueUrl: aws.String(q)})
		}()

	}
	wg.Wait()

}
