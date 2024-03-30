package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go/aws"
)

type BlockRequest struct {
	BlockNumber int `json:"blockNumber"`
}

type BlockData struct {
	BlockNumber int    `json:"blockNumber"`
	Data        string `json:"data"`
	Status      string `json:"status"`
}

func Handler(ctx context.Context, snsEvent events.SNSEvent) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	ddbClient := dynamodb.NewFromConfig(cfg)
	snsClient := sns.NewFromConfig(cfg)

	for _, record := range snsEvent.Records {
		var request BlockRequest
		fmt.Println("Message:", record.SNS.Message)
		err := json.Unmarshal([]byte(record.SNS.Message), &request)
		if err != nil {
			fmt.Println("Failed to unmarshal block data", err)
			return err
		}

		// For Simplicity generate a block on a fly

		blockSample := BlockData{
			BlockNumber: request.BlockNumber,
			Data:        "Block Data",
			Status:      "Success",
		}

		item, err := attributevalue.MarshalMap(blockSample)
		if err != nil {
			fmt.Println("Failed to marshal block data", err)
			continue
		}

		_, err = ddbClient.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(os.Getenv("DDB_TABLE_NAME")),
			Item:      item,
		})

		if err != nil {
			fmt.Println("Failed to put item", err)
			continue
		}

		responseMessage, err := json.Marshal(blockSample)
		if err != nil {
			fmt.Printf("Failed to marshal response message: %s\n", err)
			continue
		}

		_, err = snsClient.Publish(ctx, &sns.PublishInput{
			Message:  aws.String(string(responseMessage)),
			TopicArn: aws.String(os.Getenv("DATA_READY_TOPIC_ARN")),
		})
		if err != nil {
			fmt.Printf("Failed to publish response message to SNS: %s\n", err)
			continue
		}

	}
	return nil

}

func main() {
	lambda.Start(Handler)
}
