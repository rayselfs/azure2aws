package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/user/azure2aws/internal/saml"
)

func AssumeRoleWithSAML(role *saml.AWSRole, samlAssertion string, durationSeconds int32, region, output string) (*Credentials, error) {
	ctx := context.Background()

	if region == "" {
		region = "us-east-1"
	}

	cfg := aws.Config{
		Region: region,
	}

	stsClient := sts.NewFromConfig(cfg)

	input := &sts.AssumeRoleWithSAMLInput{
		RoleArn:         aws.String(role.RoleARN),
		PrincipalArn:    aws.String(role.PrincipalARN),
		SAMLAssertion:   aws.String(samlAssertion),
		DurationSeconds: aws.Int32(durationSeconds),
	}

	result, err := stsClient.AssumeRoleWithSAML(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to assume role: %w", err)
	}

	if result.Credentials == nil {
		return nil, fmt.Errorf("no credentials returned from AssumeRoleWithSAML")
	}

	creds := &Credentials{
		AccessKeyID:     aws.ToString(result.Credentials.AccessKeyId),
		SecretAccessKey: aws.ToString(result.Credentials.SecretAccessKey),
		SessionToken:    aws.ToString(result.Credentials.SessionToken),
		Expiration:      aws.ToTime(result.Credentials.Expiration),
		Region:          region,
		Output:          output,
	}

	if result.AssumedRoleUser != nil {
		creds.AssumedRoleARN = aws.ToString(result.AssumedRoleUser.Arn)
	}

	return creds, nil
}

func GetSessionDuration(configuredDuration int, samlDuration int64) int32 {
	if configuredDuration > 0 {
		return int32(configuredDuration)
	}
	if samlDuration > 0 {
		return int32(samlDuration)
	}
	return 3600
}

func IsExpired(expiration time.Time) bool {
	return time.Until(expiration) < 5*time.Minute
}
