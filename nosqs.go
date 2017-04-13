// +build !sqs

package main

import (
	"context"
	"log"
)

func runSQSListener(ctx context.Context) {
	log.Println("Simple Builder SQS Listener: excluded at compile time.")
}
