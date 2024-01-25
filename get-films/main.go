package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	"log"
	"os"
	"strings"
)

var sess = session.Must(session.NewSession())
var db = dynamodb.New(sess)

var recommendationTemplateBeginning = "Recommend me 5 films, do not ask me questions, just generate film ideas."
var recommendationTemplateJson = "\nDo not write me anything except JSON. Give it to me in the following JSON format:\n[\n{\n\"name\": \"filmName\",\n\"year\": 2013,\n\"genres\":[\"genre1\",\"genre2\"],\n\"directedBy\":\"director\",\n\"description\":\"description\"\n}\n]"

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

	messageContent := constructMessageContent(result)

	//chatGpt request
	client := openai.NewClient(os.Getenv("OpenAIToken"))
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are an experienced cinema critique.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: messageContent,
				},
			},
		},
	)
	if err != nil {
		log.Fatalf("ChatCompletion error: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "ChatCompletion error: " + err.Error(),
		}, err
	}

	content := resp.Choices[0].Message.Content
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       content,
	}, nil
}

func getUserIdAndVerify(req events.APIGatewayProxyRequest) (string, error) {
	userId, err := uuid.Parse(req.QueryStringParameters["id"])

	return userId.String(), err
}

func constructMessageContent(result *dynamodb.GetItemOutput) string {
	var messageContent string
	if result.Item != nil {
		var excludedFilms []string
		var unlikedFilms []string
		var likedFilms []string
		for _, v := range result.Item["unlikedFilms"].L {
			if v.S != nil {
				excludedFilms = append(excludedFilms, *v.S)
				unlikedFilms = append(unlikedFilms, *v.S)
			}
		}
		for _, v := range result.Item["likedFilms"].L {
			if v.S != nil {
				excludedFilms = append(excludedFilms, *v.S)
				likedFilms = append(likedFilms, *v.S)
			}
		}

		messageContent = recommendationTemplateBeginning
		if len(likedFilms) > 0 {
			messageContent = messageContent + "\nI like the following films: " + strings.Join(likedFilms, ", ") + "."
		}
		if len(unlikedFilms) > 0 {
			messageContent = messageContent + "\nI do not like the following films: " + strings.Join(unlikedFilms, ", ") + "."
		}

		messageContent = messageContent + "\nExclude the following films: " + strings.Join(excludedFilms, ", ") + recommendationTemplateJson
	} else {
		messageContent = recommendationTemplateBeginning + recommendationTemplateJson
	}

	return messageContent
}
