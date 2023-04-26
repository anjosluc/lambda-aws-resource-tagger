package main

import (
	"fmt"
	"encoding/json"
	"context"
	"log"
	"os"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type ConfigNonCompliantEventDetail struct {
	ResourceId	   		string `json:"resourceId"`
	Region         		string `json:"awsRegion"`
	AccountId      		string `json:"awsAccountId"`
	ResourceType   		string `json:"resourceType"`
	ConfigRuleName 		string `json:"configRuleName"`
	NewEvaluationResult struct {
		ComplianceType  string `json:"complianceType"`
	} `json:"newEvaluationResult"`
}

type SQSEventBodyDetail struct {
	Detail 		json.RawMessage `json:"detail"`
}

func getSession(region string) (*session.Session, error) {
	sess, err := session.NewSession(
		&aws.Config{
			Region: aws.String(region),
		},
	)

	if err != nil {
		log.Fatal("Could not create AWS session: ", err)
		return nil, err
	}

	return sess, nil
}

func getARN(session *session.Session, configEvent ConfigNonCompliantEventDetail) (string, error) {
	svc := configservice.New(session)
	
	batchGetConfigInput := &configservice.BatchGetResourceConfigInput{
		ResourceKeys: []*configservice.ResourceKey {
			{
				ResourceId: aws.String(configEvent.ResourceId),
				ResourceType: aws.String(configEvent.ResourceType),
			},
		},
	}

	resourceConfig, err := svc.BatchGetResourceConfig(batchGetConfigInput)
	
	if err != nil {
		log.Fatal("Could not get Resource Config: ", err)
	}

	//ASSUMING THAT IS JUST ONE NON COMPLIANT ITEM PER INVOCATION
	if len(resourceConfig.BaseConfigurationItems) <= 0 {
		log.Fatal("Could not get ARN, probably resource ", configEvent.ResourceId, " - ", configEvent.ResourceType, " has been deleted")
	}

	arn := *resourceConfig.BaseConfigurationItems[0].Arn
	return arn, nil
}

func tagARN(session *session.Session, arn string) (bool, error) {
	svc := resourcegroupstaggingapi.New(session)

	tagResourcesInput := &resourcegroupstaggingapi.TagResourcesInput{
		ResourceARNList: aws.StringSlice([]string{arn}),
		Tags: map[string]*string {
			os.Getenv("BUSINESS_TAG_KEY"): aws.String(os.Getenv("BUSINESS_TAG_VALUE")),
		},
	}

	output, err := svc.TagResources(tagResourcesInput)
	
	if err != nil {
		log.Fatal("Could not tag Resource: ", err)
	} else if len(output.FailedResourcesMap) > 0 {
		log.Fatal("Could not tag Resources: ", output.FailedResourcesMap)
	}

	fmt.Println("ARN ", arn, " successfully tagged!")
	return true, nil
}

func handleRequest(ctx context.Context, event events.SQSEvent) (bool, error) {
	var bodyMap SQSEventBodyDetail
	message := event.Records[0]
	bodyRaw := json.RawMessage(message.Body)
	bodyBytes, err := json.Marshal(bodyRaw)
	fmt.Println("Message ", message.Body)
	
	if err != nil {
        panic(err)
    }

	json.Unmarshal(bodyBytes, &bodyMap)

	var configEventDetail ConfigNonCompliantEventDetail
	err = json.Unmarshal(bodyMap.Detail, &configEventDetail)

	if err != nil {
		log.Fatal("Could not unmarshal scheduled event: ", err)
		fmt.Println("Could not unmarshal scheduled event: ", err)
	}
	
	session, err := getSession(configEventDetail.Region)
	objectARN, err := getARN(session, configEventDetail)
	
	fmt.Println("ARN to tag: ", objectARN)
	isTagged, err := tagARN(session, objectARN)
	
	if err != nil || isTagged != true {
		return false, nil
	} else {
		return true, nil
	}
	
}

func main() {
	lambda.Start(handleRequest)
}