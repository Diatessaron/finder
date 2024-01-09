package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/sashabaranov/go-openai"
	"log"
	"os"
	"strings"
)

var db = dynamodb.New(session.Must(session.NewSession()))

func main() {
	lambda.Start(handleRequest)
}

func handleRequest(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	//err := godotenv.Load(".env")
	//if err != nil {
	//	log.Fatalf("Error loading .env file")
	//}

	userEmail := req.QueryStringParameters["userEmail"]

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

	var messageContent string
	if result.Item != nil {
		var excludedFilms []string
		unlikedFilms := result.Item["unlikedFilms"]
		likedFilms := result.Item["likedFilms"]
		for _, v := range unlikedFilms.L {
			if v.S != nil {
				excludedFilms = append(excludedFilms, *v.S)
			}
		}
		for _, v := range likedFilms.L {
			if v.S != nil {
				excludedFilms = append(excludedFilms, *v.S)
			}
		}

		messageContent = "Recommend me 10 films, do not ask me questions, just generate film ideas.\nExclude the following films: " + strings.Join(excludedFilms, ", ") + ".\nDo not write me anything except JSON. Give it to me in the following JSON format:\n[\n{\n\"name\": \"Her\",\n\"year\": 2013,\n\"genres\":[\"Sci-Fi\",\"Romance\"],\n\"directedBy\":\"Spike Jonze\",\n\"description\":\"this film is a unique and touching exploration of love and relationships in the age of technology. It tells the story of a lonely writer who develops an unlikely relationship with an artificially intelligent operating system\"\n}\n]"
	} else {
		messageContent = "Recommend me 10 films, do not ask me questions, just generate film ideas.\nDo not write me anything except JSON. Give it to me in the following JSON format:\n[\n{\n\"name\": \"Her\",\n\"year\": 2013,\n\"genres\":[\"Sci-Fi\",\"Romance\"],\n\"directedBy\":\"Spike Jonze\",\n\"description\":\"this film is a unique and touching exploration of love and relationships in the age of technology. It tells the story of a lonely writer who develops an unlikely relationship with an artificially intelligent operating system\"\n}\n]"
	}

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
	fmt.Println(content)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       content,
	}, nil

	//var movies []Movie
	//err = json.Unmarshal([]byte(content), &movies)
	//if err != nil {
	//	log.Fatalf("Error parsing JSON: %s", err)
	//}
	//fmt.Println(movies)
}

//type Movie struct {
//	Name        string   `json:"name"`
//	Year        int      `json:"year"`
//	Genres      []string `json:"genres"`
//	DirectedBy  string   `json:"directedBy"`
//	Description string   `json:"description"`
//}
