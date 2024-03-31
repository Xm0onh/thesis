package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type BlockRequest struct {
	BlockNumber int    `json:"blockNumber"`
	RequestorID string `json:"requestorID"`
}

type BlockData struct {
	BlockNumber int    `json:"blockNumber"`
	Data        string `json:"data"`
	Status      string `json:"status"`
}

func HandleRequest(ctx context.Context, event json.RawMessage) (string, error) {
	var snsEvent events.SNSEvent
	if err := json.Unmarshal(event, &snsEvent); err != nil && len(snsEvent.Records) >= 0 {
		return handleResponseProcessing(ctx, snsEvent)
	}

	var blockRequest BlockRequest
	err := json.Unmarshal(event, &blockRequest)

	if err != nil {
		return handleRequestInitiation(ctx, blockRequest)
	}

	return "Invalid event type", fmt.Errorf("invalid event type")
}

func handleResponseProcessing(ctx context.Context, snsEvent events.SNSEvent) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "Failed to load AWS configuration", err
	}

	ddbClient := dynamodb.NewFromConfig(cfg)

	for _, record := range snsEvent.Records {
		var blockData BlockRequest
		err := json.Unmarshal([]byte(record.SNS.Message), &blockData)
		if err != nil {
			return "Failed to unmarshal block data", err
		}

		out, err := ddbClient.GetItem(ctx, &dynamodb.GetItemInput{
			TableName: aws.String(os.Getenv("DDB_TABLE_NAME")),
			Key: map[string]types.AttributeValue{
				"ID":          &types.AttributeValueMemberS{Value: blockData.RequestorID},
				"BlockNumber": &types.AttributeValueMemberN{Value: strconv.Itoa(blockData.BlockNumber)},
			},
		})

		if err != nil {
			return "Failed to get item from DynamoDB", err
		}

		err = attributevalue.UnmarshalMap(out.Item, &blockData)
		if err != nil {
			return "Failed to unmarshal item from DynamoDB", err
		}

		fmt.Printf("Fetched data for block #%d: %+v\n", blockData.BlockNumber, blockData)
	}

	return "Successfully processed response messages", nil
}

func handleRequestInitiation(ctx context.Context, blockRequest BlockRequest) (string, error) {

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "Failed to marshal request message", err
	}

	snsClient := sns.NewFromConfig(cfg)
	requestTopicARN := os.Getenv("DATA_READY_TOPIC_ARN")
	requestMessage, err := json.Marshal(blockRequest)

	if err != nil {
		return "Failed to marshal request message", err
	}

	_, _ = snsClient.Publish(ctx, &sns.PublishInput{
		Message:  aws.String(string(requestMessage)),
		TopicArn: &requestTopicARN,
	})

	return fmt.Sprintf("Block request for block number %d has been published to SNS topic", blockRequest.BlockNumber), nil
}

func main() {
	lambda.Start(HandleRequest)
}
