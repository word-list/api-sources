package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

type Source struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Url  string `json:"url"`
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	switch request.HTTPMethod {
	case http.MethodGet:
		return getHandler(request)
	case http.MethodPost:
		return postHandler(request)
	case http.MethodPut:
		return putHandler(request)
	case http.MethodDelete:
		return deleteHandler(request)
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusMethodNotAllowed,
			Body:       "unsupported http method",
		}, nil
	}
}

func getDynamoDbService() (*dynamodb.Client, error) {

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
		return nil, err
	}

	svc := dynamodb.NewFromConfig(cfg)
	return svc, nil
}

func getSourceFromRecord(rec map[string]types.AttributeValue) Source {
	return Source{
		ID:   rec["id"].(*types.AttributeValueMemberS).Value,
		Name: rec["name"].(*types.AttributeValueMemberS).Value,
		Url:  rec["url"].(*types.AttributeValueMemberS).Value,
	}
}

func getAllHandler() (events.APIGatewayProxyResponse, error) {

	svc, err := getDynamoDbService()
	if err != nil {
		log.Printf("Failed to get DynamoDB service")
		return serverError("Failed to retrieve sources")
	}

	input := &dynamodb.ScanInput{
		TableName: aws.String(os.Getenv("SOURCES_TABLE_NAME")),
	}

	result, err := svc.Scan(context.TODO(), input)
	if err != nil {
		log.Printf("Failed to retrieve items: %v", err)
		return serverError("Failed to retrieve sources")
	}

	sources := make([]Source, len(result.Items))
	for i := range result.Items {
		sources[i] = getSourceFromRecord(result.Items[i])
	}

	responseBody, _ := json.Marshal(sources)

	return ok(string(responseBody))
}

func getHandler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	id := req.PathParameters["id"]
	if id == "" {
		// return all
		return getAllHandler()
	}

	svc, err := getDynamoDbService()
	if err != nil {
		log.Printf("Failed to get DynamoDB service")
		return serverError("Failed to retrieve source")
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("SOURCES_TABLE_NAME")),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	}

	result, err := svc.GetItem(context.TODO(), input)
	if err != nil {
		log.Printf("Failed to get item: %v", err)
		return serverError("Failed to retrieve source")
	}

	if result.Item == nil {
		return events.APIGatewayProxyResponse{StatusCode: 404, Body: `{"error": "not found"}`}, nil
	}

	source := getSourceFromRecord(result.Item)
	responseBody, _ := json.Marshal(source)

	return ok(string(responseBody))
}

func serverError(message string) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       fmt.Sprintf(`{"error": "%s"}`, message),
	}, nil
}

func ok(content string) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       content,
	}, nil
}

func created(content string) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Body:       content,
	}, nil
}

func postHandler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var source Source
	json.Unmarshal([]byte(req.Body), &source)

	svc, err := getDynamoDbService()
	if err != nil {
		log.Printf("Failed to get DynamoDB service")
		return serverError("Failed to create source")
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("SOURCES_TABLE_NAME")),
		Item: map[string]types.AttributeValue{
			"id":   &types.AttributeValueMemberS{Value: uuid.New().String()},
			"name": &types.AttributeValueMemberS{Value: source.Name},
			"url":  &types.AttributeValueMemberS{Value: source.Url},
		},
	}

	_, err = svc.PutItem(context.TODO(), input)
	if err != nil {
		log.Printf("Failed to create item: %v", err)
		return serverError("Failed to store source")
	}

	return created(source.ID)
}

func putHandler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var source Source
	json.Unmarshal([]byte(req.Body), &source)

	svc, err := getDynamoDbService()
	if err != nil {
		log.Printf("Failed to get DynamoDB service")
		return serverError("Failed to update source")
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("SOURCES_TABLE_NAME")),
		Item: map[string]types.AttributeValue{
			"id":   &types.AttributeValueMemberS{Value: source.ID},
			"name": &types.AttributeValueMemberS{Value: source.Name},
			"url":  &types.AttributeValueMemberS{Value: source.Url},
		},
	}

	_, err = svc.PutItem(context.TODO(), input)
	if err != nil {
		log.Printf("Failed to update item: %v", err)
		return serverError("Failed to store source")
	}

	return ok(source.ID)
}

func deleteHandler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	id := req.PathParameters["id"]
	if id == "" {
		// return all
		return getAllHandler()
	}

	svc, err := getDynamoDbService()
	if err != nil {
		log.Printf("Failed to get DynamoDB service")
		return serverError("Failed to retrieve source")
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(os.Getenv("SOURCES_TABLE_NAME")),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	}

	_, err = svc.DeleteItem(context.TODO(), input)
	if err != nil {
		log.Printf("Failed to delete item: %v", err)
		return serverError("Failed to delete source")
	}

	return ok(id)
}

func main() {
	lambda.Start(handler)
}
