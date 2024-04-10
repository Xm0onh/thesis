package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	blockchainPkg "github.com/xm0onh/thesis/packages/blockchain"
	lubyTransform "github.com/xm0onh/thesis/packages/luby"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
)

// ///
// //// Setup the LTBlock struct
var exp = 5

var blockchain = initializeBlockchain()

func BlockToByte(block []*blockchainPkg.Block) []byte {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(block)
	if err != nil {
		log.Fatalf("failed to encode block: %v", err)
	}
	return buffer.Bytes()
}

func ByteToBlock(data []byte) *[]blockchainPkg.Block {
	var block []blockchainPkg.Block
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)
	err := decoder.Decode(&block)
	if err != nil {
		log.Fatalf("failed to decode block: %v", err)
	}
	return &block
}

func BlockchainToBytes(bc *blockchainPkg.Blockchain) []byte {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(bc)
	if err != nil {
		log.Fatalf("failed to encode blockchain: %v", err)
	}
	return buffer.Bytes()
}

func BytesToBlockchain(data []byte) *blockchainPkg.Blockchain {
	var bc blockchainPkg.Blockchain
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)
	err := decoder.Decode(&bc)
	if err != nil {
		log.Fatalf("failed to decode blockchain: %v", err)
	}
	return &bc
}

func initializeBlockchain() *blockchainPkg.Blockchain {
	bc := &blockchainPkg.Blockchain{}

	// Simulate transaction data
	transactions := blockchainPkg.GenerateTransactionsForBlock()
	// Add blocks to the blockchain
	for i := 0; i < exp; i++ { // You can adjust the number of blocks
		block := blockchainPkg.CreateBlock(i, transactions)
		bc.AddBlock(block)
	}

	return bc
}

func GenerateDroplet(blockNumber []int) ([]lubyTransform.LTBlock, int) {
	fmt.Println("hey there from droplets")
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	if err := enc.Encode(blockchain); err != nil {
		fmt.Printf("Error encoding object: %s\n", err)
		return nil, 0
	}
	var tempBlockchain []*blockchainPkg.Block
	for _, blockNumber := range blockNumber {
		tempBlockchain = append(tempBlockchain, &blockchain.Chain[blockNumber])
	}
	message := BlockToByte(tempBlockchain)
	fmt.Println("Message: ", len(message))

	// Define parameters for the Luby Codec.
	sourceBlocks := int(exp * 15 / 10)
	degreeCDF := lubyTransform.SolitonDistribution(sourceBlocks)

	// Create a PRNG source.
	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)

	// Create a new Luby Codec.
	codec := lubyTransform.NewLubyCodec(sourceBlocks, random, degreeCDF)

	// Encode the message into LTBlocks.
	// Commitment size
	encodedBlockIDs := make([]int64, int(exp*20/10))
	for i := range encodedBlockIDs {
		encodedBlockIDs[i] = int64(i)
	}

	droplets := lubyTransform.EncodeLTBlocks(message, encodedBlockIDs, codec)

	if err := enc.Encode(droplets); err != nil {
		fmt.Printf("Error encoding object: %s\n", err)
		return nil, 0
	}

	return droplets, len(message)
}

///// End of LTBlock struct setup

/////

type LTBlock struct {
	BlockCode int64  `json:"blockCode"`
	Data      []byte `json:"data"`
}

type SizeOfMessage struct {
	MessageSize int `json:"messageSize"`
}
type requestedBlock struct {
	BlockNumber []int `json:"blockNumber"`
}

var ddbTableName = os.Getenv("DDB_TABLE_NAME")

func init() {
	gob.Register(blockchainPkg.Transaction{})
	gob.Register(blockchainPkg.Block{})
}

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
		droplets, messageSize := GenerateDroplet(block.BlockNumber)

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
