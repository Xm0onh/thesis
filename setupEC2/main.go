package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	blockchainPkg "github.com/xm0onh/thesis/packages/blockchain"
	lubyTransform "github.com/xm0onh/thesis/packages/luby"
	utils "github.com/xm0onh/thesis/packages/utils"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/kzg"
	fiatshamir "github.com/consensys/gnark-crypto/fiat-shamir"
)

var tableName = "setup"
var bucketName = "thesisubc"

func init() {
	gob.Register(blockchainPkg.Transaction{})
	gob.Register(blockchainPkg.Block{})
	gob.Register(kzg.OpeningProof{})
	gob.Register(bn254.G1Affine{})
	gob.Register(fr.Element{})
}

func main() {
	ctx := context.TODO()
	var requestedBlocks []int

	for i := 0; i < 999; i++ {
		requestedBlocks = append(requestedBlocks, i)
	}

	// Here you need to configure your event manually or simulate one
	event := utils.StartSignal{
		Start:           true,
		SourceBlocks:    1000,
		EncodedBlockIDs: 2000,
		NumberOfBlocks:  1000,
		RequestedBlocks: requestedBlocks,
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-west-1"))
	if err != nil {
		log.Fatalf("Unable to load SDK config, %v", err)
	}

	ddbClient := dynamodb.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)

	degreeCDF := lubyTransform.SolitonDistribution(event.SourceBlocks)
	degreeCDFString, _ := json.Marshal(degreeCDF)

	seed := time.Now().UnixNano()
	blockchain := utils.InitializeBlockchain(event.NumberOfBlocks, 1000)

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
	srs := SetupKZG()
	var droplets = utils.GenerateDroplet(SetupParameters)
	hashes := HashDroplets(droplets)
	digest, point, proof := Prover(srs, hashes)

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
			"srs":             &types.AttributeValueMemberB{Value: SerializeSRS(srs)},
			"digest":          &types.AttributeValueMemberB{Value: digest.Marshal()},
			"point":           &types.AttributeValueMemberB{Value: point.Marshal()},
			"proof":           &types.AttributeValueMemberB{Value: SerializeOpeningProof(proof)},
		},
	})
	if err != nil {
		fmt.Printf("Failed to put metadata item into DynamoDB: %v\n", err)
		return
	}

	fmt.Println("Process completed successfully !")
}

// uploadToS3 uploads a byte slice to an S3 bucket under the specified key
func uploadToS3(ctx context.Context, s3Client *s3.Client, bucket, key string, data []byte) error {
	_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})

	return err
}

func SetupKZG() *kzg.SRS {
	alpha := big.NewInt(42)
	srs, err := kzg.NewSRS(2048, alpha)
	if err != nil {
		fmt.Println("Error creating SRS:", err)
		return nil
	}
	return srs
}

func HashDroplets(droplets []lubyTransform.LTBlock) []fr.Element {
	hashes := make([]fr.Element, len(droplets))
	for i, droplet := range droplets {
		hasher := sha256.New()
		hasher.Write(droplet.Data)
		hash := hasher.Sum(nil)
		hashes[i].SetBytes(hash)
	}
	return hashes
}

func Prover(srs *kzg.SRS, hashes []fr.Element) (bn254.G1Affine, fr.Element, kzg.OpeningProof) {
	digest, err := kzg.Commit(hashes, srs.Pk)
	if err != nil {
		fmt.Println("Error committing to hashes:", err)
		return bn254.G1Affine{}, fr.Element{}, kzg.OpeningProof{}
	}
	transcript := fiatshamir.NewTranscript(sha256.New())
	digestBytes := digest.Marshal()
	transcript.Bind("commitment_digest", digestBytes)
	challengeBytes, _ := transcript.ComputeChallenge("evaluation_point")
	var point fr.Element
	point.SetBytes(challengeBytes)
	proof, err := kzg.Open(hashes, point, srs.Pk)
	if err != nil {
		fmt.Println("Open error:", err)
		return bn254.G1Affine{}, fr.Element{}, kzg.OpeningProof{}
	}
	return digest, point, proof
}

func SerializeSRS(srs *kzg.SRS) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(srs)
	if err != nil {
		return nil
	}
	return buf.Bytes()
}

func DeserializeSRS(data []byte) (*kzg.SRS, error) {
	var srs kzg.SRS
	buf := bytes.NewReader(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&srs)
	if err != nil {
		return nil, err
	}
	return &srs, nil
}

// SerializeOpeningProof serializes the kzg.OpeningProof using gob
func SerializeOpeningProof(proof kzg.OpeningProof) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(proof)
	if err != nil {
		return nil
	}
	return buf.Bytes()
}

// DeserializeOpeningProof deserializes the kzg.OpeningProof from a byte slice
func DeserializeOpeningProof(data []byte) (kzg.OpeningProof, error) {
	var proof kzg.OpeningProof
	buf := bytes.NewReader(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&proof)
	if err != nil {
		return kzg.OpeningProof{}, err
	}
	return proof, nil
}
