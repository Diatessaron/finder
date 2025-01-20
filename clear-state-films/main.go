package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/google/uuid"
	"log"
)

var sess = session.Must(session.NewSession())
var db = dynamodb.New(sess)

func main() {
	lambda.Start(handleRequest)
}

func handleRequest(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userId, err := getUserIdAndVerify(req)
	if err != nil {
		id := req.QueryStringParameters["id"]
		log.Fatalf("Provided user id is not correct, user id - %s", id)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Provided user id is not correct, user id - " + id,
		}, err
	}

	db.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String("user_films"),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(userId),
			},
		},
	})
	if err != nil {
		log.Fatalf("Got error calling DeleteItemInput: %s", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Got error calling DeleteItemInput: " + err.Error(),
		}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 204,
	}, nil
}

func getUserIdAndVerify(req events.APIGatewayProxyRequest) (string, error) {
	userId, err := uuid.Parse(req.QueryStringParameters["id"])

	return userId.String(), err
}
