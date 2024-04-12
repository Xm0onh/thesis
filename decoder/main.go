package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	blockchainPkg "github.com/xm0onh/thesis/packages/blockchain"
	lubyTransform "github.com/xm0onh/thesis/packages/luby"
	utils "github.com/xm0onh/thesis/packages/utils"
)

var setupTableName = os.Getenv("SETUP_DB")

var Droplets []lubyTransform.LTBlock

var ddbClient *dynamodb.Client
var tableName = os.Getenv("DDB_TABLE_NAME")

func init() {
	gob.Register(blockchainPkg.Transaction{})
	gob.Register(blockchainPkg.Block{})
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}
	ddbClient = dynamodb.NewFromConfig(cfg)
}

func Handler(ctx context.Context, snsEvent events.SNSEvent) ([]blockchainPkg.Block, error) {
	fmt.Println("Received notification from SNS, downloading items from DynamoDB")

	pag := dynamodb.NewScanPaginator(ddbClient, &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})

	for pag.HasMorePages() {
		out, err := pag.NextPage(ctx)
		if err != nil {
			fmt.Printf("Failed to scan DynamoDB table: %v\n", err)
			return []blockchainPkg.Block{}, err
		}

		for _, item := range out.Items {
			var block utils.LTBlock
			err := attributevalue.UnmarshalMap(item, &block)
			if err != nil {
				fmt.Printf("Failed to unmarshal DynamoDB item to LTBlock: %v\n", err)
				continue
			}
			if len(block.Data) != 0 {
				Droplets = append(Droplets, lubyTransform.LTBlock{
					BlockCode: block.BlockCode,
					Data:      block.Data,
				})
			}
		}
	}

	messageSize, err := utils.FetchMessageSize(ctx, tableName, ddbClient)
	if err != nil {
		log.Printf("Error fetching message size: %v", err)
		return []blockchainPkg.Block{}, err
	}
	param := utils.SetupParameters{}
	param.DegreeCDF, param.SourceBlocks, param.EncodedBlockIDs, param.RandomSeed, param.NumberOfBlocks, err = utils.PullDataFromSetup(ctx, setupTableName)
	if err != nil {
		fmt.Printf("Failed to pull data from setup: %v\n", err)
		return []blockchainPkg.Block{}, err
	}
	fmt.Printf("Downloaded %d LTBlocks.\n", len(Droplets))
	return utils.Decoder(Droplets, messageSize, param)
}

func main() {

	lambda.Start(Handler)
}
