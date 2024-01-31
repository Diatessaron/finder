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

	filmToRemove := req.QueryStringParameters["filmToRemove"]

	//retrieving user info from DynamoDB
	result, err := db.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String("user_films"),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(userId),
			},
		},
	})
	if err != nil {
		log.Fatalf("Got error calling GetItem: %s", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Got error calling GetItem: " + err.Error(),
		}, err
	}

	if result.Item == nil {
		log.Fatalf("Got error calling GetItem, user with id %s not found", userId)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Got error calling GetItem, user not found.",
		}, err
	}

	//remove film from likedFilms and resave others
	oldLikedFilms := result.Item["likedFilms"].L
	filmToRemoveIndex := -1
	for index, film := range oldLikedFilms {
		if *film.S == filmToRemove {
			filmToRemoveIndex = index
		}
	}
	if filmToRemoveIndex == -1 {
		log.Fatalf("Got error removing film - %s. Film not found", filmToRemove)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Got error removing film - " + ". Film not found.",
		}, err
	}
	resultLikedFilms := append(oldLikedFilms[:filmToRemoveIndex], oldLikedFilms[filmToRemoveIndex+1:]...)

	_, err = db.UpdateItem(&dynamodb.UpdateItemInput{
		TableName: aws.String("user_films"),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(userId),
			},
		},
		ConditionExpression: aws.String("attribute_not_exists(likedFilms) OR likedFilms = :oldVal"),
		UpdateExpression:    aws.String("SET likedFilms = :val"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":val": {
				L: resultLikedFilms,
			},
			":oldVal": {
				L: oldLikedFilms,
			},
		},
		ReturnValues: aws.String("UPDATED_NEW"),
	})
	if err != nil {
		log.Fatalf("Got error calling PutItem: %s", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Got error calling PutItem: " + err.Error(),
		}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Success",
	}, nil
}

func getUserIdAndVerify(req events.APIGatewayProxyRequest) (string, error) {
	userId, err := uuid.Parse(req.QueryStringParameters["id"])

	return userId.String(), err
}
