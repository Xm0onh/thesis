package main

import (
	"bytes"
	"context"
	"crypto/sha256"
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

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/kzg"
	fiatshamir "github.com/consensys/gnark-crypto/fiat-shamir"

	blockchainPkg "github.com/xm0onh/thesis/packages/blockchain"
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
	gob.Register(kzg.OpeningProof{})
	gob.Register(bn254.G1Affine{})
	gob.Register(fr.Element{})

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
	// Decoding the blocks
	startTime := time.Now()
	utils.Decoder(Droplets, param)
	fmt.Println("Successfully Decoded the blocks.")
	fmt.Println("Time to decode: ", time.Since(startTime))
	// verification

	srs, digest, point, proof, err := PullKZGData(ctx, setupTableName)
	if err != nil {
		fmt.Printf("Failed to pull KZG data: %v\n", err)
		return false, err
	}
	fmt.Println(Verifier(digest, proof, point, &srs.Vk))

	/// Submit the time to the time keeper table
	_, err = ddbClient.PutItem(ctx, &dynamodb.PutItemInput{
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

	return true, nil
}

func PullKZGData(ctx context.Context, setupTableName string) (
	srs *kzg.SRS,
	digest bn254.G1Affine,
	point fr.Element,
	proof kzg.OpeningProof,
	err error,
) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, bn254.G1Affine{}, fr.Element{}, kzg.OpeningProof{}, fmt.Errorf("failed to load AWS configuration: %v", err)
	}

	ddbClient := dynamodb.NewFromConfig(cfg)
	result, err := ddbClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(setupTableName),
		Key: map[string]types.AttributeValue{
			"ID": &types.AttributeValueMemberS{Value: "setup"},
		},
	})
	if err != nil {
		return nil, bn254.G1Affine{}, fr.Element{}, kzg.OpeningProof{}, fmt.Errorf("failed to get item from DynamoDB: %v", err)
	}

	// Deserialize SRS
	srsData := result.Item["srs"].(*types.AttributeValueMemberB).Value
	srs, err = DeserializeSRS(srsData)
	if err != nil {
		return nil, bn254.G1Affine{}, fr.Element{}, kzg.OpeningProof{}, fmt.Errorf("failed to deserialize SRS: %v", err)
	}

	// Deserialize Digest
	digestData := result.Item["digest"].(*types.AttributeValueMemberB).Value
	digest = bn254.G1Affine{}
	digest.Unmarshal(digestData)

	if err != nil {
		return nil, bn254.G1Affine{}, fr.Element{}, kzg.OpeningProof{}, fmt.Errorf("failed to deserialize Digest: %v", err)
	}

	// Deserialize Point
	pointData := result.Item["point"].(*types.AttributeValueMemberB).Value
	point.Unmarshal(pointData)

	if err != nil {
		return nil, bn254.G1Affine{}, fr.Element{}, kzg.OpeningProof{}, fmt.Errorf("failed to deserialize Point: %v", err)
	}

	// Deserialize Proof
	proofData := result.Item["proof"].(*types.AttributeValueMemberB).Value
	proof, err = DeserializeOpeningProof(proofData)
	if err != nil {
		return nil, bn254.G1Affine{}, fr.Element{}, kzg.OpeningProof{}, fmt.Errorf("failed to deserialize Proof: %v", err)
	}
	return
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

func Verifier(digest bn254.G1Affine, proof kzg.OpeningProof, point fr.Element, vk *kzg.VerifyingKey) bool {
	transcript := fiatshamir.NewTranscript(sha256.New())
	digestBytes := digest.Marshal()
	transcript.Bind("commitment_digest", digestBytes)
	challengeBytes, _ := transcript.ComputeChallenge("evaluation_point")
	var verifierPoint fr.Element
	verifierPoint.SetBytes(challengeBytes)
	if !verifierPoint.Equal(&point) {
		fmt.Println("Computed point does not match the proof point")
		return false
	}
	return kzg.Verify(&digest, &proof, verifierPoint, *vk) == nil
}

func main() {

	lambda.Start(Handler)
}
