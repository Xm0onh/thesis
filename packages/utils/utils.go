package utils

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	blockchainPkg "github.com/xm0onh/thesis/packages/blockchain"
	lubyTransform "github.com/xm0onh/thesis/packages/luby"
)

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

func InitializeBlockchain(NumberOfBlocks int) *blockchainPkg.Blockchain {
	bc := &blockchainPkg.Blockchain{}

	// Simulate transaction data
	transactions := blockchainPkg.GenerateTransactionsForBlock()
	// Add blocks to the blockchain
	for i := 0; i < NumberOfBlocks; i++ {
		block := blockchainPkg.CreateBlock(i, transactions)
		bc.AddBlock(block)
	}

	return bc
}

func CalculateMessageAndMessageSize(blockchain blockchainPkg.Blockchain, blockNumber []int) ([]byte, int, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	if err := enc.Encode(blockchain); err != nil {
		fmt.Printf("Error encoding object: %s\n", err)
		return []byte{0}, 0, err
	}
	var tempBlockchain []*blockchainPkg.Block
	for _, blockNumber := range blockNumber {
		tempBlockchain = append(tempBlockchain, &blockchain.Chain[blockNumber])
	}
	message := BlockToByte(tempBlockchain)
	return message, len(message), nil
}

func PullDataFromSetup(ctx context.Context, setupTableName string) (
	degreeCDF []float64,
	sourceBlocks,
	encodedBlockIDs int,
	randomSeed int64,
	numberOfBlocks int,
	message []byte,
	messageSize int,
	err error) {

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Printf("failed to load AWS configuration, %v\n", err)
		return
	}

	ddbClient := dynamodb.NewFromConfig(cfg)

	result, err := ddbClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(setupTableName),
		Key: map[string]types.AttributeValue{
			"ID": &types.AttributeValueMemberS{Value: "setup"},
		},
	})
	if err != nil {
		fmt.Printf("failed to get item from DynamoDB: %v\n", err)
		return
	}

	// Extracting DegreeCDF
	if v, ok := result.Item["degreeCDF"].(*types.AttributeValueMemberS); ok {
		err = json.Unmarshal([]byte(v.Value), &degreeCDF)
		if err != nil {
			fmt.Printf("error parsing degreeCDF: %v\n", err)
			return
		}
	}

	// Extracting SourceBlocks
	if v, ok := result.Item["sourceBlocks"].(*types.AttributeValueMemberN); ok {
		sourceBlocks, err = strconv.Atoi(v.Value)
		if err != nil {
			fmt.Printf("error parsing sourceBlocks: %v\n", err)
			return
		}
	}

	// Extracting RandomSeed
	if v, ok := result.Item["randomSeed"].(*types.AttributeValueMemberN); ok {
		var rs int64
		rs, err = strconv.ParseInt(v.Value, 10, 64)
		if err != nil {
			fmt.Printf("error parsing randomSeed: %v\n", err)
			return
		}
		randomSeed = int64(rs) // Note: Be cautious with potential int64 to int conversion issues on 32-bit systems
	}

	// Extracting EncodedBlockIDs
	if v, ok := result.Item["encodedBlockIDs"].(*types.AttributeValueMemberN); ok {
		encodedBlockIDs, err = strconv.Atoi(v.Value)
		if err != nil {
			fmt.Printf("error parsing encodedBlockIDs: %v\n", err)
			return
		}
	}

	// Extracting NumberOfBlocks
	if v, ok := result.Item["numberOfBlocks"].(*types.AttributeValueMemberN); ok {
		numberOfBlocks, err = strconv.Atoi(v.Value)
		if err != nil {
			fmt.Printf("error parsing numberOfBlocks: %v\n", err)
			return
		}
	}

	// Extracting Message
	if v, ok := result.Item["message"].(*types.AttributeValueMemberB); ok {
		message = v.Value
	}

	// Extracting MessageSize
	if v, ok := result.Item["messageSize"].(*types.AttributeValueMemberN); ok {
		messageSize, err = strconv.Atoi(v.Value)
		if err != nil {
			fmt.Printf("error parsing messageSize: %v\n", err)
			return
		}
	}

	return
}

func GenerateDroplet(param SetupParameters) []lubyTransform.LTBlock {
	fmt.Println("hey there from droplets")
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	// Define parameters for the Luby Codec.
	sourceBlocks := param.SourceBlocks
	degreeCDF := param.DegreeCDF

	// Create a PRNG source.

	seedValue := param.RandomSeed
	seed := rand.NewSource(seedValue)
	random := rand.New(seed)

	// Create a new Luby Codec.
	codec := lubyTransform.NewLubyCodec(sourceBlocks, random, degreeCDF)

	// Encode the message into LTBlocks.
	// Commitment size
	encodedBlockIDs := make([]int64, param.EncodedBlockIDs)
	for i := range encodedBlockIDs {
		encodedBlockIDs[i] = int64(i)
	}

	droplets := lubyTransform.EncodeLTBlocks(param.Message, encodedBlockIDs, codec)

	if err := enc.Encode(droplets); err != nil {
		fmt.Printf("Error encoding object: %s\n", err)
		return nil
	}

	return droplets
}

func Decoder(Droplets []lubyTransform.LTBlock, param SetupParameters) ([]blockchainPkg.Block, error) {
	// message := "Hello, World!"
	sourceBlocks := param.SourceBlocks
	degreeCDF := param.DegreeCDF

	// Create a PRNG source.
	seedValue := param.RandomSeed
	seed := rand.NewSource(seedValue)
	random := rand.New(seed)

	// Create a new Luby Codec.
	codec := lubyTransform.NewLubyCodec(sourceBlocks, random, degreeCDF)

	decoder := codec.NewDecoder(param.MessageSize)

	if decoder.AddBlocks(Droplets) {
		decodedMessage := decoder.Decode()

		if decodedMessage != nil {
			// Convert blockchain bytes to a Blocks object.
			decodedBlocks := ByteToBlock(decodedMessage)
			// size of decodedBlockchain
			fmt.Println("Decoded blockchain: ", decodedBlocks)
			return *decodedBlocks, nil

		} else {
			fmt.Println("Not enough blocks to decode the message.")
		}
	} else {
		fmt.Println("Not enough blocks to decode the message.")
	}
	return []blockchainPkg.Block{}, nil
}
