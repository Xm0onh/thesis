package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	blockchainPkg "github.com/xm0onh/thesis/packages/blockchain"
	kzg "github.com/xm0onh/thesis/packages/kzg"
	lubyTransform "github.com/xm0onh/thesis/packages/luby"
	utils "github.com/xm0onh/thesis/packages/utils"
)

var tableName = "setup"
var bucketName = "thesisubc"

func init() {
	gob.Register(blockchainPkg.Transaction{})
	gob.Register(blockchainPkg.Block{})
}

func main() {
	ctx := context.Background()

	// Here you need to configure your event manually or simulate one
	event := utils.StartSignal{
		Start:           true,
		SourceBlocks:    100,            // Set appropriately
		EncodedBlockIDs: 10,             // Set appropriately
		NumberOfBlocks:  50,             // Set appropriately
		RequestedBlocks: []int{1, 2, 3}, // Sample requested blocks
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-west-1"),
	)
	if err != nil {
		log.Fatalf("Unable to load SDK config, %v", err)
	}

	ddbClient := dynamodb.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)

	degreeCDF := lubyTransform.SolitonDistribution(event.SourceBlocks)
	degreeCDFString, _ := json.Marshal(degreeCDF)

	seed := time.Now().UnixNano()
	blockchain := utils.InitializeBlockchain(event.NumberOfBlocks, 100)

	message, messageSize, err := utils.CalculateMessageAndMessageSize(*blockchain, event.RequestedBlocks)
	if err != nil {
		fmt.Printf("Failed to evaluate message size: %v\n", err)
		return
	}

	objectKey := "blockchain_data"
	err = uploadToS3(ctx, s3Client, bucketName, objectKey, message)
	if err != nil {
		fmt.Printf("Failed to upload message to S3: %v\n", err)
		return
	}

	SetupParameters := utils.SetupParameters{
		DegreeCDF:       degreeCDF,
		RandomSeed:      seed,
		SourceBlocks:    event.SourceBlocks,
		EncodedBlockIDs: event.EncodedBlockIDs,
		NumberOfBlocks:  event.NumberOfBlocks,
		MessageSize:     messageSize,
		Message:         message,
	}

	droplets := utils.GenerateDroplet(SetupParameters)
	kzg.CalculateKZGParam(ctx, bucketName, droplets)

	_, err = ddbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]types.AttributeValue{
			"ID":              &types.AttributeValueMemberS{Value: "setup"},
			"degreeCDF":       &types.AttributeValueMemberS{Value: string(degreeCDFString)},
			"randomSeed":      &types.AttributeValueMemberN{Value: strconv.FormatInt(seed, 10)},
			"sourceBlocks":    &types.AttributeValueMemberN{Value: strconv.Itoa(event.SourceBlocks)},
			"encodedBlockIDs": &types.AttributeValueMemberN{Value: strconv.Itoa(event.EncodedBlockIDs)},
			"numberOfBlocks":  &types.AttributeValueMemberN{Value: strconv.Itoa(event.NumberOfBlocks)},
			"requestedBlocks": &types.AttributeValueMemberS{Value: fmt.Sprint(event.RequestedBlocks)},
			"messageSize":     &types.AttributeValueMemberN{Value: strconv.Itoa(messageSize)},
			"S3ObjectKey":     &types.AttributeValueMemberS{Value: objectKey},
		},
	})
	if err != nil {
		fmt.Printf("Failed to put metadata item into DynamoDB: %v\n", err)
		return
	}

	fmt.Println("Process completed successfully!")
}

// uploadToS3 uploads a byte slice to an S3 bucket under the specified key
func uploadToS3(ctx context.Context, client *s3.Client, bucket, key string, data []byte) error {
	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	return err
}
