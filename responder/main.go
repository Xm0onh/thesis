package main

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	blockchainPkg "github.com/xm0onh/thesis/packages/blockchain"
	utils "github.com/xm0onh/thesis/packages/utils"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
)

var ddbTableName = os.Getenv("DDB_TABLE_NAME")
var setupTableName = os.Getenv("SETUP_DB")

func init() {
	gob.Register(blockchainPkg.Transaction{})
	gob.Register(blockchainPkg.Block{})
}

func Handler(ctx context.Context, snsEvent events.SNSEvent) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS configuration, %w", err)
	}
	param := utils.SetupParameters{}
	param.DegreeCDF, param.SourceBlocks, param.EncodedBlockIDs, param.RandomSeed, param.NumberOfBlocks, _ = utils.PullDataFromSetup(ctx)
	if err != nil {
		fmt.Printf("Failed to pull data from setup: %v\n", err)
		return err
	}
	ddbClient := dynamodb.NewFromConfig(cfg)
	blockchain := utils.InitializeBlockchain(param.NumberOfBlocks)
	for _, record := range snsEvent.Records {
		var block utils.RequestedBlocks
		err := json.Unmarshal([]byte(record.SNS.Message), &block)
		fmt.Println("Received request for block numbers: ", block.BlockNumber)
		if err != nil {
			fmt.Printf("Failed to unmarshal LTBlock data: %v\n", err)
			continue
		}
		droplets, messageSize := utils.GenerateDroplet(*blockchain, block.BlockNumber, param)

		// Add MessageSize into the Database
		_, err = ddbClient.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(ddbTableName),
			Item: map[string]types.AttributeValue{
				"ID":          &types.AttributeValueMemberS{Value: "message"},
				"MessageSize": &types.AttributeValueMemberN{Value: strconv.Itoa(messageSize)},
			},
		})
		if err != nil {
			fmt.Printf("Failed to put metadata item into DynamoDB: %v\n", err)
			return err
		}

		for _, droplet := range droplets {

			_, err = ddbClient.PutItem(ctx, &dynamodb.PutItemInput{
				TableName: aws.String(ddbTableName),
				Item: map[string]types.AttributeValue{
					"ID":        &types.AttributeValueMemberS{Value: strconv.FormatInt(droplet.BlockCode, 10)},
					"Data":      &types.AttributeValueMemberB{Value: droplet.Data},
					"BlockCode": &types.AttributeValueMemberN{Value: strconv.FormatInt(droplet.BlockCode, 10)},
				},
				ConditionExpression: aws.String("attribute_not_exists(ID)"),
			})

			if err != nil {
				fmt.Printf("Failed to put LTBlock item into DynamoDB: %v\n", err)
				continue
			}
		}

	}

	return nil
}

func main() {
	lambda.Start(Handler)
}
