package main

import (
	"context"
	"encoding/json"
	"finder/tmdb"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	"log"
	"os"
	"strconv"
	"strings"
)

var sess = session.Must(session.NewSession())
var db = dynamodb.New(sess)

var recommendationTemplateBeginning = "Recommend me exactly {filmCount} film."
var recommendationTemplateEnding = "\nDo not include mentioned films.\nProvide me response in the json form of an array of strings with name 'films'."

func main() {
	lambda.Start(handleRequest)
}

func handleRequest(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	filmCount := getFilmCount(req)
	userId, err := getUserIdAndVerify(req)
	filmsToExclude := req.MultiValueQueryStringParameters["filmsToExclude"]
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

	messageContent := constructMessageContent(result, filmCount, filmsToExclude)
	log.Printf("Prompt, message content to ChatGPT - %s", messageContent)

	//chatGpt request
	client := openai.NewClient(os.Getenv("OpenAIToken"))
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			},
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleSystem,
					Content: "You are an expert in film recommendations and an experienced cinema critique designed to output JSON. " +
						"You recommend films, do not ask questions, just generate film ideas, write only film names. I give you films I like and films I do not like. " +
						"Also I give you films I do not want to see in your film recommendation list. Based on this, you will generate me film ideas.",
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

	var filmRecommendationsObject FilmRecommendations
	err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), &filmRecommendationsObject)
	if err != nil {
		log.Println(resp.Choices[0].Message.Content)
		log.Fatalf("Error while parsing ChatGPT response as an object. Error message - %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error while parsing ChatGPT response: " + err.Error(),
		}, err
	}
	filmRecommendationsArray := filmRecommendationsObject.Films

	log.Printf("Film recommendations: %v\n", filmRecommendationsArray)

	films, err := tmdb.NormalizeFilms(filmRecommendationsArray)
	if err != nil {
		log.Fatalf("Error while parsing ChatGPT response. Error message - %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error while normalizing films. Error - " + err.Error(),
		}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       films,
	}, nil
}

func getUserIdAndVerify(req events.APIGatewayProxyRequest) (string, error) {
	userId, err := uuid.Parse(req.QueryStringParameters["id"])

	return userId.String(), err
}

func getFilmCount(req events.APIGatewayProxyRequest) string {
	filmCount := req.QueryStringParameters["filmCount"]
	if filmCount == "" {
		return "5"
	}

	filmCountInt, err := strconv.Atoi(filmCount)
	if filmCountInt <= 0 || err != nil {
		return "5"
	}

	return filmCount
}

func constructMessageContent(result *dynamodb.GetItemOutput, filmCount string, filmsToExclude []string) string {
	var messageContent string
	if result.Item != nil {
		excludedFilms := filmsToExclude
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

		messageContent = messageContent + "\nExclude the following films: " + strings.Join(excludedFilms, ", ") + recommendationTemplateEnding
	} else {
		messageContent = recommendationTemplateBeginning + recommendationTemplateEnding
	}

	return strings.ReplaceAll(messageContent, "{filmCount}", filmCount)
}

type FilmRecommendations struct {
	Films []string `json:"films"`
}
