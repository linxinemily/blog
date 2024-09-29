package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"google.golang.org/api/indexing/v3"
	"google.golang.org/api/option"
)

type Body struct {
	Payload Payload `json:"payload"`
}

type Payload struct {
	Url   string `json:"url"`
	Title string `json:"title"`
}

func handler(request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	fullUrl, err := getNewArticleFullUrlFromReqBody(request.Body)
	if err != nil {
		return &events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       err.Error(),
		}, nil
	}

	credentialJsonStr := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	ctx := context.Background()
	indexingService, err := indexing.NewService(ctx, option.WithCredentialsJSON([]byte(credentialJsonStr)))

	if err != nil {
		log.Fatalln(err)
	}

	notification := &indexing.UrlNotification{
		Url:  fullUrl,
		Type: "URL_UPDATED",
	}
	_, err = indexingService.UrlNotifications.Publish(notification).Do()

	if err != nil {
		log.Fatalln(err)
	}

	metaRes, err := indexingService.UrlNotifications.
		GetMetadata().
		Url(fullUrl).
		Do()

	log.Printf("%s %s %s\n", metaRes.LatestUpdate.NotifyTime, metaRes.LatestUpdate.Type, metaRes.LatestUpdate.Url)

	return &events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       fmt.Sprintf("Success send request to notify Google of new article, url: %s", fullUrl),
	}, nil
}

func getNewArticleFullUrlFromReqBody(reqBody string) (string, error) {
	var body Body
	_ = json.Unmarshal([]byte(reqBody), &body)
	r, _ := regexp.Compile("(^new article:)(.*)")

	if !r.MatchString(body.Payload.Title) || r.FindStringSubmatch(body.Payload.Title)[2] == "" {
		return "", errors.New("no article should be index")
	}

	slug := strings.Trim(r.FindStringSubmatch(body.Payload.Title)[2], " ")
	var fullUrl strings.Builder
	fullUrl.WriteString(body.Payload.Url)
	fullUrl.WriteString("/posts/")
	fullUrl.WriteString(slug)

	return fullUrl.String(), nil
}

func main() {
	lambda.Start(handler)
}
