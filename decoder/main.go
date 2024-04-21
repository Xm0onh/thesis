package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	blockchainPkg "github.com/xm0onh/thesis/packages/blockchain"
	kzg "github.com/xm0onh/thesis/packages/kzg"
	lubyTransform "github.com/xm0onh/thesis/packages/luby"
	utils "github.com/xm0onh/thesis/packages/utils"
)

var setupTableName = os.Getenv("SETUP_DB")
var tableName = os.Getenv("DDB_TABLE_NAME")
var timeKeeperTable = os.Getenv("TIME_KEEPER_TABLE")
var bucketName = os.Getenv("BLOCKCHAIN_S3_BUCKET")

var Droplets []lubyTransform.LTBlock
var ddbClient *dynamodb.Client

func init() {
	gob.Register(blockchainPkg.Transaction{})
	gob.Register(blockchainPkg.Block{})
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}
	ddbClient = dynamodb.NewFromConfig(cfg)
}

func Handler(ctx context.Context, snsEvent events.SNSEvent) (bool, error) {
	fmt.Println("Received notification from SNS, downloading items from DynamoDB")

	pag := dynamodb.NewScanPaginator(ddbClient, &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})

	for pag.HasMorePages() {
		out, err := pag.NextPage(ctx)
		if err != nil {
			fmt.Printf("Failed to scan DynamoDB table: %v\n", err)
			return false, err
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

	param := utils.SetupParameters{}
	param.DegreeCDF, param.SourceBlocks, param.EncodedBlockIDs, param.RandomSeed, param.NumberOfBlocks, _, param.MessageSize, _ = utils.PullDataFromSetup(ctx, setupTableName)
	fmt.Printf("Downloaded %d LTBlocks.\n", len(Droplets))

	/// Submit the time to the time keeper table
	_, err := ddbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(timeKeeperTable),
		Item: map[string]types.AttributeValue{
			"ID":        &types.AttributeValueMemberS{Value: "decoder"},
			"Timestamp": &types.AttributeValueMemberS{Value: time.Now().Format("2006-01-02T15:04:05.999999")},
		},
	})
	if err != nil {
		fmt.Printf("Failed to submit time to time keeper table: %v\n", err)
		return false, err
	}
	//decoder

	// Decoding the blocks
	startTime := time.Now()
	utils.Decoder(Droplets, param)
	fmt.Println("Successfully Decoded the blocks.")
	fmt.Println("Time to decode: ", time.Since(startTime))
	// verification
	startTime = time.Now()
	kzg.Verification(ctx, bucketName)
	fmt.Println("Time to verify: ", time.Since(startTime))

	return true, nil
}

func main() {

	lambda.Start(Handler)
}
