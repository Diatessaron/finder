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

var db = dynamodb.New(session.Must(session.NewSession()))

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

	userLikedFilm := []*dynamodb.AttributeValue{}
	userUnlikedFilm := []*dynamodb.AttributeValue{}

	method := req.QueryStringParameters["method"]
	film := req.QueryStringParameters["film"]
	if method == "like" {
		userLikedFilm = append(userLikedFilm, &dynamodb.AttributeValue{S: aws.String(film)})
	} else if method == "unlike" {
		userUnlikedFilm = append(userUnlikedFilm, &dynamodb.AttributeValue{S: aws.String(film)})
	}

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

	//create or update user liked/unliked films
	var resultLikedFilms []*dynamodb.AttributeValue
	var resultUnlikedFilms []*dynamodb.AttributeValue
	oldLikedFilms := []*dynamodb.AttributeValue{}
	oldUnlikedFilms := []*dynamodb.AttributeValue{}
	if result.Item == nil {
		resultLikedFilms = userLikedFilm
		resultUnlikedFilms = userUnlikedFilm
	} else {
		resultLikedFilms = append(userLikedFilm, result.Item["likedFilms"].L...)
		resultUnlikedFilms = append(userUnlikedFilm, result.Item["unlikedFilms"].L...)
		oldLikedFilms = result.Item["likedFilms"].L
		oldUnlikedFilms = result.Item["unlikedFilms"].L
	}

	_, err = db.UpdateItem(&dynamodb.UpdateItemInput{
		TableName: aws.String("user_films"),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(userId),
			},
		},
		ConditionExpression: aws.String("attribute_not_exists(likedFilms) OR attribute_not_exists(unlikedFilms) " +
			"OR likedFilms = :oldLikedFilms OR unlikedFilms = :oldUnlikedFilms"),
		UpdateExpression: aws.String("SET likedFilms = :likedFilms, unlikedFilms = :unlikedFilms"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":oldLikedFilms": {
				L: oldLikedFilms,
			},
			":oldUnlikedFilms": {
				L: oldUnlikedFilms,
			},
			":likedFilms": {
				L: resultLikedFilms,
			},
			":unlikedFilms": {
				L: resultUnlikedFilms,
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
