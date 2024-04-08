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
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type LTBlock struct {
	BlockCode int64  `json:"blockCode"`
	Data      []byte `json:"data"`
}

type requestedBlock struct {
	BlockNumber int `json:"blockNumber"`
}

var ddbTableName = os.Getenv("DDB_TABLE_NAME")

func Handler(ctx context.Context, snsEvent events.SNSEvent) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS configuration, %w", err)
	}

	ddbClient := dynamodb.NewFromConfig(cfg)

	for _, record := range snsEvent.Records {
		var block requestedBlock
		err := json.Unmarshal([]byte(record.SNS.Message), &block)
		fmt.Println("Received request for block number: ", block.BlockNumber)
		if err != nil {
			fmt.Printf("Failed to unmarshal LTBlock data: %v\n", err)
			continue
		}
		droplets := GenerateDroplet()
		_, err = ddbClient.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(ddbTableName),
			Item: map[string]types.AttributeValue{
				"ID":        &types.AttributeValueMemberS{Value: strconv.FormatInt(droplets[0].BlockCode, 10)},
				"Data":      &types.AttributeValueMemberB{Value: droplets[0].Data},
				"BlockCode": &types.AttributeValueMemberN{Value: strconv.FormatInt(droplets[0].BlockCode, 10)},
			},
			ConditionExpression: aws.String("attribute_not_exists(ID)"),
		})

		if err != nil {
			fmt.Printf("Failed to put LTBlock item into DynamoDB: %v\n", err)
			continue
		}
	}

	return nil
}

func main() {
	lambda.Start(Handler)
}
