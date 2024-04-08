package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"math/rand"
	blockchainPkg "reqReceiver/pkg/blockchain"
	lubyTransform "reqReceiver/pkg/luby"
	"time"
)

// //// Setup the LTBlock struct
var exp = 5

var blockchain = initializeBlockchain()

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

func GenerateDroplet() []lubyTransform.LTBlock {

	gob.Register(blockchainPkg.Transaction{})
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	if err := enc.Encode(blockchain); err != nil {
		fmt.Printf("Error encoding object: %s\n", err)
		return nil
	}

	message := BlockToByte(&blockchain.Chain[0])

	// Define parameters for the Luby Codec.
	sourceBlocks := int(exp * 11 / 10)
	degreeCDF := lubyTransform.SolitonDistribution(sourceBlocks)

	// Create a PRNG source.
	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)

	// Create a new Luby Codec.
	codec := lubyTransform.NewLubyCodec(sourceBlocks, random, degreeCDF)

	// Encode the message into LTBlocks.
	// Commitment size
	encodedBlockIDs := make([]int64, int(exp*13/10))
	for i := range encodedBlockIDs {
		encodedBlockIDs[i] = int64(i)
	}

	droplets := lubyTransform.EncodeLTBlocks(message, encodedBlockIDs, codec)

	if err := enc.Encode(droplets); err != nil {
		fmt.Printf("Error encoding object: %s\n", err)
		return nil
	}

	return droplets
}

///// End of LTBlock struct setup
