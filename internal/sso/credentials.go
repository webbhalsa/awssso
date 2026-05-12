package sso

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
)

type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	ExpiresAt       time.Time
}

func GetRoleCredentials(ctx context.Context, region, accessToken, accountID, roleName string) (*Credentials, error) {
	cfg, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := sso.NewFromConfig(cfg)
	out, err := client.GetRoleCredentials(ctx, &sso.GetRoleCredentialsInput{
		AccessToken: aws.String(accessToken),
		AccountId:   aws.String(accountID),
		RoleName:    aws.String(roleName),
	})
	if err != nil {
		return nil, fmt.Errorf("get role credentials: %w", err)
	}

	c := out.RoleCredentials
	return &Credentials{
		AccessKeyID:     aws.ToString(c.AccessKeyId),
		SecretAccessKey: aws.ToString(c.SecretAccessKey),
		SessionToken:    aws.ToString(c.SessionToken),
		ExpiresAt:       time.UnixMilli(c.Expiration),
	}, nil
}
