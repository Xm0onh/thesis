package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

var decoderStarterSNS = os.Getenv("DECODER_INIT_SNS_TOPIC_ARN")
var counterTable = os.Getenv("COUNTER_TABLE_NAME")
var commitmentSize = os.Getenv("COMMITMENT_SIZE")

var ddbClient *dynamodb.Client
var snsClient *sns.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}
	ddbClient = dynamodb.NewFromConfig(cfg)
	snsClient = sns.NewFromConfig(cfg)
}

func Handler(ctx context.Context, ddbEvent events.DynamoDBEvent) error {
	fmt.Println("Received DynamoDB event")
	for _, record := range ddbEvent.Records {
		if record.EventName == "INSERT" {
			updatedCounter := incrementCounter(ctx)
			commitmentSizeInt, err := strconv.ParseInt(commitmentSize, 10, 64)
			if err != nil {
				fmt.Printf("Failed to parse commitment size: %v\n", err)
				return err
			}
			if updatedCounter == commitmentSizeInt {
				_, err := snsClient.Publish(ctx, &sns.PublishInput{
					Message:  aws.String(commitmentSize + "items reached in DynamoDB table"),
					TopicArn: aws.String(decoderStarterSNS),
				})
				if err != nil {
					fmt.Printf("Error publishing to SNS: %v\n", err)
					return err
				}
				fmt.Println("Published message to SNS topic.")
			}
		}
	}
	return nil
}

func incrementCounter(ctx context.Context) int64 {
	out, err := ddbClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(counterTable),
		Key: map[string]types.AttributeValue{
			"ID": &types.AttributeValueMemberS{Value: "Counter"},
		},
		UpdateExpression: aws.String("SET #Count = if_not_exists(#Count, :start) + :inc"),
		ExpressionAttributeNames: map[string]string{
			"#Count": "Count",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":inc":   &types.AttributeValueMemberN{Value: "1"},
			":start": &types.AttributeValueMemberN{Value: "0"},
		},
		ReturnValues: types.ReturnValueUpdatedNew,
	})
	if err != nil {
		fmt.Printf("Failed to update counter: %v\n", err)
		return -1
	}

	newCountStr := out.Attributes["Count"].(*types.AttributeValueMemberN).Value
	newCount, err := strconv.ParseInt(newCountStr, 10, 64)
	if err != nil {
		fmt.Printf("Failed to parse new counter value: %v\n", err)
		return -1
	}

	return newCount
}

func main() {
	lambda.Start(Handler)
}
