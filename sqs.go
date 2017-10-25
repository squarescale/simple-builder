// +build sqs

package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/squarescale/simple-builder/build"
)

const sqsPollTime = 20
const sqsHideTime = 60
const sqsHideRefresh = 30

func runSQSListener(ctx context.Context) {
	input_sqs := os.Getenv("SIMPLE_BUILDER_INPUT_SQS")
	if input_sqs == "" {
		log.Println("Simple Builder SQS Listener: missing SIMPLE_BUILDER_INPUT_SQS environment to start")
		return
	}

	log.Println("Starting Simple Builder SQS Listener...")
	defer log.Println("Stopped Simple Builder SQS Listener.")

	creds := credentials.NewEnvCredentials()
	sess, err := session.NewSession(aws.NewConfig().WithRegion(os.Getenv("AWS_REGION")).WithCredentials(creds))
	if err != nil {
		log.Printf("SQS[%s] error establishing session:", input_sqs, err)
		return
	}

	svc := sqs.New(sess)

	for ctx.Err() == nil {

		errors := make(chan error, 1)
		responses := make(chan *sqs.ReceiveMessageOutput, 1)

		go func() {
			resp, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(input_sqs),
				MaxNumberOfMessages: aws.Int64(1),
				WaitTimeSeconds:     aws.Int64(sqsPollTime),
				VisibilityTimeout:   aws.Int64(sqsHideTime),
			})
			if err != nil {
				errors <- err
			} else {
				responses <- resp
			}
		}()

		select {
		case <-ctx.Done():
			return
		case err := <-errors:
			log.Printf("SQS[%s]: error receiving messages: %s", input_sqs, err)
		case resp := <-responses:
			for _, msg := range resp.Messages {
				if ctx.Err() != nil {
					break
				}
				go sqsHandleMessage(ctx, input_sqs, svc, msg)
			}
		}
	}
}

func sqsHandleMessage(ctx context.Context, input_sqs string, svc *sqs.SQS, msg *sqs.Message) {
	var err error

	log.Printf("SQS[%s]: Handle message %s", input_sqs, *msg.ReceiptHandle)

	var buildDescriptor struct {
		build.BuildDescriptor
		Callbacks []string `json:"callbacks"`
	}
	err = json.Unmarshal([]byte(*msg.Body), &buildDescriptor)
	if err != nil {
		log.Printf("SQS[%s]: error unmarshaling message %s: %s", input_sqs, *msg.ReceiptHandle, err)
		return
	}

	b, tk, err := buildsHandler.CreateBuild(buildDescriptor.BuildDescriptor, buildDescriptor.Callbacks)
	if err != nil {
		log.Printf("SQS[%s]: error creating build for message %s: %s", input_sqs, *msg.ReceiptHandle, err)
		return
	}

	log.Printf("[build %s] imported from sqs message %s", tk, *msg.ReceiptHandle)

	for ctx.Err() == nil {
		tmout, _ := context.WithTimeout(ctx, sqsHideRefresh*time.Second)
		select {
		case <-ctx.Done():
		case <-b.Done():
			log.Printf("[build %s] delete SQS message", tk)
			_, err = svc.DeleteMessage(&sqs.DeleteMessageInput{
				QueueUrl:      aws.String(input_sqs),
				ReceiptHandle: msg.ReceiptHandle,
			})
			if err != nil {
				log.Printf("[build %s] SQS[%s]: error deleting message message %s: %s", tk, input_sqs, *msg.ReceiptHandle, err)
				return
			}
			return
		case <-tmout.Done():
			log.Printf("[build %s] hide SQS message for %d seconds", tk, sqsHideTime)
			_, err = svc.ChangeMessageVisibility(&sqs.ChangeMessageVisibilityInput{
				QueueUrl:          aws.String(input_sqs),
				ReceiptHandle:     msg.ReceiptHandle,
				VisibilityTimeout: aws.Int64(sqsHideTime),
			})
			if err != nil {
				log.Printf("[build %s] SQS[%s]: error deleting message message %s: %s", tk, input_sqs, *msg.ReceiptHandle, err)
				return
			}
		}
	}

}
