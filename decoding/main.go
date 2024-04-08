package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type LTBlock struct {
	BlockCode int64  `json:"blockCode"`
	Data      []byte `json:"data"`
}

// Droplets holds the LTBlocks downloaded during the current invocation
var Droplets []LTBlock

var ddbClient *dynamodb.Client
var tableName = os.Getenv("DDB_TABLE_NAME")

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}
	ddbClient = dynamodb.NewFromConfig(cfg)
}

func Handler(ctx context.Context, snsEvent events.SNSEvent) error {
	fmt.Println("Received notification from SNS, downloading items from DynamoDB")

	// Reset Droplets for the current invocation
	Droplets = []LTBlock{}

	pag := dynamodb.NewScanPaginator(ddbClient, &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})

	for pag.HasMorePages() {
		out, err := pag.NextPage(ctx)
		if err != nil {
			fmt.Printf("Failed to scan DynamoDB table: %v\n", err)
			return err
		}

		for _, item := range out.Items {
			var block LTBlock
			err := attributevalue.UnmarshalMap(item, &block)
			if err != nil {
				fmt.Printf("Failed to unmarshal DynamoDB item to LTBlock: %v\n", err)
				continue // Skip this item and continue with the next.
			}
			Droplets = append(Droplets, block)
		}
	}

	fmt.Printf("Downloaded %d LTBlocks.\n", len(Droplets))
	// Here, you can process Droplets as needed.

	return nil
}

func main() {
	lambda.Start(Handler)
}
