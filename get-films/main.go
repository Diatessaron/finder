package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/dynamodb"
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
	userEmail, err := getUserEmail(req)
	if err != nil {
		log.Fatalf("Got error calling getUserEmail: %s", err)
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

func getUserEmail(req events.APIGatewayProxyRequest) (string, error) {
	authHeader := req.Headers["authorization"]
	accessToken := strings.TrimPrefix(authHeader, "Bearer ")

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

	return userEmail, nil
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
