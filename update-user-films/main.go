package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"log"
	"strings"
)

var db = dynamodb.New(session.Must(session.NewSession()))

func main() {
	lambda.Start(handleRequest)
}

func handleRequest(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userEmail := getUserEmail(req)
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
			"email": {
				S: aws.String(userEmail),
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
	if result.Item == nil {
		resultLikedFilms = userLikedFilm
		resultUnlikedFilms = userUnlikedFilm
	} else {
		resultLikedFilms = append(userLikedFilm, result.Item["likedFilms"].L...)
		resultUnlikedFilms = append(userUnlikedFilm, result.Item["unlikedFilms"].L...)
	}

	item := map[string]*dynamodb.AttributeValue{
		"email": {
			S: aws.String(userEmail),
		},
		"likedFilms": {
			L: resultLikedFilms,
		},
		"unlikedFilms": {
			L: resultUnlikedFilms,
		},
	}

	_, err = db.PutItem(&dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String("user_films"),
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

func getUserEmail(req events.APIGatewayProxyRequest) string {
	authHeader := req.Headers["authorization"]
	accessToken := strings.TrimPrefix(authHeader, "Bearer ")

	sess := session.Must(session.NewSession())
	svc := cognitoidentityprovider.New(sess)

	input := &cognitoidentityprovider.GetUserInput{
		AccessToken: aws.String(accessToken),
	}

	user, _ := svc.GetUser(input)
	userEmail := ""
	for _, attr := range user.UserAttributes {
		if *attr.Name == "email" {
			userEmail = *attr.Value
			break
		}
	}

	return userEmail
}
