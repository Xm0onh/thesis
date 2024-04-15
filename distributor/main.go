package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	lambdasdk "github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

// Message represents the JSON structure to send to each SNS topic.
type Message struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

func invokeTimeKeeper(lambdaClient *lambdasdk.Client) {

	// Payload could be modified or left empty as per your requirements
	payload := []byte(`{}`)
	result, err := lambdaClient.Invoke(context.TODO(), &lambdasdk.InvokeInput{
		FunctionName: aws.String("arn:aws:lambda:us-west-1:428232754477:function:TimeKeeper"),
		Payload:      payload,
	})
	if err != nil {
		fmt.Printf("Error invoking TimeKeeper Lambda function: %v\n", err)
		return
	}
	fmt.Printf("Successfully invoked TimeKeeper Lambda function with response: %s\n", string(result.Payload))
}

func publishMessage(snsClient *sns.Client, topicArn string, msg Message, wg *sync.WaitGroup) {
	defer wg.Done()

	// Acquire a token
	// sem <- struct{}{}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		fmt.Printf("Error marshaling message: %v\n", err)
		// <-sem
		return
	}

	_, err = snsClient.Publish(context.TODO(), &sns.PublishInput{
		Message:  aws.String(string(msgBytes)),
		TopicArn: aws.String(topicArn),
	})
	if err != nil {
		fmt.Printf("Error publishing to SNS topic %s: %v\n", topicArn, err)
	} else {
		fmt.Printf("Published to %s: %s\n", topicArn, msgBytes)
	}

	// Release the token
	// <-sem
}

// Start event
type StartEvent struct {
	Start bool `json:"start"`
}

func handler(ctx context.Context, event StartEvent) {
	if !event.Start {
		fmt.Println("Not a start event")
		return
	}

	fmt.Println("Starting the distribution")
	totalResponders := 50
	totalRange := 600
	intervalSize := totalRange / totalResponders
	// batchSize := 25

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Printf("Error loading AWS configuration: %v\n", err)
		return
	}
	lambdaClient := lambdasdk.NewFromConfig(cfg)
	snsClient := sns.NewFromConfig(cfg)
	invokeTimeKeeper(lambdaClient)
	var wg sync.WaitGroup
	// semaphore := make(chan struct{}, batchSize)

	for i := 1; i <= totalResponders; i++ {
		responderEnv := fmt.Sprintf("RESPONDER_%d", i)
		topicArn := os.Getenv(responderEnv)
		if topicArn == "" {
			fmt.Printf("Environment variable %s is not set\n", responderEnv)
			continue
		}

		msg := Message{
			Start: i * intervalSize,
			End:   (i + 1) * intervalSize,
		}

		wg.Add(1)
		go publishMessage(snsClient, topicArn, msg, &wg)
	}

	wg.Wait() // Wait for all goroutines to complete
}

func main() {
	lambda.Start(handler)
}
