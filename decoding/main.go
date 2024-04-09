package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	blockchainPkg "github.com/xm0onh/thesis/packages/blockchain"
	lubyTransform "github.com/xm0onh/thesis/packages/luby"
)

var exp = 5

// //
func BlockToByte(block *blockchainPkg.Block) []byte {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(block)
	if err != nil {
		log.Fatalf("failed to encode block: %v", err)
	}
	return buffer.Bytes()
}

func ByteToBlock(data []byte) *blockchainPkg.Block {
	var block blockchainPkg.Block
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)
	err := decoder.Decode(&block)
	if err != nil {
		log.Fatalf("failed to decode block: %v", err)
	}
	return &block
}

func Decoder(Droplets []lubyTransform.LTBlock, messageSize int) (blockchainPkg.Block, error) {
	// message := "Hello, World!"
	sourceBlocks := int(exp * 15 / 10)
	degreeCDF := lubyTransform.SolitonDistribution(sourceBlocks)

	// Create a PRNG source.
	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)

	// Create a new Luby Codec.
	codec := lubyTransform.NewLubyCodec(sourceBlocks, random, degreeCDF)

	decoder := codec.NewDecoder(messageSize)

	if decoder.AddBlocks(Droplets) {
		// If the decoder has enough blocks, try to decode the message.
		decodedMessage := decoder.Decode()

		if decodedMessage != nil {
			// Convert blockchain bytes to a Blockchain object.
			decodedBlock := ByteToBlock(decodedMessage)
			// size of decodedBlockchain
			fmt.Println("Decoded blockchain: ", decodedBlock)
			return *decodedBlock, nil

		} else {
			fmt.Println("Not enough blocks to decode the message.")
		}
	} else {
		fmt.Println("Not enough blocks to decode the message.")
	}
	return blockchainPkg.Block{}, nil
}

// /
type LTBlock struct {
	BlockCode int64  `json:"blockCode"`
	Data      []byte `json:"data"`
}

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

func fetchMessageSize(ctx context.Context) (int, error) {
	result, err := ddbClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"ID": &types.AttributeValueMemberS{Value: "message"},
		},
	})

	if err != nil {
		return 0, fmt.Errorf("failed to get message size from DynamoDB: %v", err)
	}

	if result.Item == nil {
		return 0, fmt.Errorf("message size item not found")
	}

	messageSizeAttr, ok := result.Item["MessageSize"]
	if !ok {
		return 0, fmt.Errorf("message size attribute not found")
	}

	messageSize, err := strconv.Atoi(messageSizeAttr.(*types.AttributeValueMemberN).Value)
	if err != nil {
		return 0, fmt.Errorf("failed to convert message size to int: %v", err)
	}

	return messageSize, nil
}

func Handler(ctx context.Context, snsEvent events.SNSEvent) (blockchainPkg.Block, error) {
	fmt.Println("Received notification from SNS, downloading items from DynamoDB")

	pag := dynamodb.NewScanPaginator(ddbClient, &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})

	for pag.HasMorePages() {
		out, err := pag.NextPage(ctx)
		if err != nil {
			fmt.Printf("Failed to scan DynamoDB table: %v\n", err)
			return blockchainPkg.Block{}, err
		}

		for _, item := range out.Items {
			var block LTBlock
			// Ensure here you correctly map DynamoDB item attributes to LTBlock struct
			err := attributevalue.UnmarshalMap(item, &block)
			if err != nil {
				fmt.Printf("Failed to unmarshal DynamoDB item to LTBlock: %v\n", err)
				continue // Skip this item and continue with the next.
			}
			if len(block.Data) != 0 {
				Droplets = append(Droplets, lubyTransform.LTBlock{
					BlockCode: block.BlockCode,
					Data:      block.Data,
				})
			}
		}
	}

	messageSize, err := fetchMessageSize(ctx)
	if err != nil {
		log.Printf("Error fetching message size: %v", err)
		return blockchainPkg.Block{}, err
	}

	fmt.Printf("Downloaded %d LTBlocks.\n", len(Droplets))
	return Decoder(Droplets, messageSize)
}

func main() {

	lambda.Start(Handler)
}
