package aws

import (
	"bytes"
	"encoding/base64"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
)

var GetECRCredentials func(accessKey, secretKey, region string) (string, string, error) = getECRCredentials

func getECRCredentials(accessKey, secretKey, region string) (string, string, error) {
	// Create an AWS session & ECR client
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
		Region:      &region,
	}))
	ecrClient := ecr.New(sess)

	// Get the ECR authorization token
	tokenOutput, err := ecrClient.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return "", "", err
	}

	// Extract the authorization token from the response
	if len(tokenOutput.AuthorizationData) == 0 {
		return "", "", err
	}
	authorizationTokenB64 := aws.StringValue(tokenOutput.AuthorizationData[0].AuthorizationToken)

	// Decode & parse authorization token
	authorizationTokenBytes, err := base64.StdEncoding.DecodeString(authorizationTokenB64)
	if err != nil {
		return "", "", err
	}
	authTokenParts := bytes.Split(authorizationTokenBytes, []byte(":"))
	username := string(authTokenParts[0])
	password := string(authTokenParts[1])

	return username, password, nil
}
