package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/google/uuid"
	"log"
	"slices"
	"sort"
	"strconv"
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

	//preparing pagination params. If no 'size' in query params, then size is totalCount
	totalCount := len(result.Item["likedFilms"].L)
	size := totalCount
	var page int
	sizeString := req.QueryStringParameters["size"]
	pageString := req.QueryStringParameters["page"]
	if sizeString != "" && pageString != "" {
		size, err = strconv.Atoi(sizeString)
		page, err = strconv.Atoi(pageString)

		if err != nil {
			log.Fatalf("Pagination params are not correct. Size - %s, Page - %s", sizeString, pageString)
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf("Pagination params are not correct. Size - %s, Page - %s, Error - %v", sizeString, pageString, err.Error()),
			}, err
		}
	}

	likedFilms := paginateFilms(result.Item["likedFilms"].L, page, size)
	likedFilms = sortFilms(likedFilms, req.QueryStringParameters["sort"])
	pageableResult := PageableResult{
		Page:       page,
		TotalCount: totalCount,
		Content:    likedFilms,
	}

	jsonArray, err := json.Marshal(pageableResult)
	if err != nil {
		log.Fatalf("Got error parsing to result JSON: %v", pageableResult)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Got error parsing string array to result JSON: " + err.Error(),
		}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(jsonArray),
	}, nil
}

type PageableResult struct {
	Page       int      `json:"page"`
	Content    []string `json:"content"`
	TotalCount int      `json:"totalCount"`
}

func getUserIdAndVerify(req events.APIGatewayProxyRequest) (string, error) {
	userId, err := uuid.Parse(req.QueryStringParameters["id"])

	return userId.String(), err
}

func paginateFilms(likedFilms []*dynamodb.AttributeValue, page int, size int) []string {
	var paginated []string

	start := page * size

	if start > len(likedFilms) {
		return paginated
	}

	end := start + size
	if end > len(likedFilms) {
		end = len(likedFilms)
	}

	for _, film := range likedFilms[start:end] {
		paginated = append(paginated, film.String())
	}

	return paginated
}

func sortFilms(likedFilms []string, sortWay string) []string {
	if sortWay == "" {
		return likedFilms
	}

	if sortWay == "ASC" {
		slices.Sort(likedFilms)
	} else if sortWay == "DESC" {
		sort.Slice(likedFilms, func(i, j int) bool {
			return likedFilms[i] > likedFilms[j]
		})
	}

	return likedFilms
}
