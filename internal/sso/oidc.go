package sso

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
)

const clientName = "awssso"

type Token struct {
	AccessToken string
	ExpiresAt   time.Time
}

// DeviceAuth performs the SSO OIDC device authorization flow and returns an access token.
func DeviceAuth(ctx context.Context, startURL, region string, openBrowser func(url string) error) (*Token, error) {
	cfg, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	oidc := ssooidc.NewFromConfig(cfg)

	reg, err := oidc.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String(clientName),
		ClientType: aws.String("public"),
	})
	if err != nil {
		return nil, fmt.Errorf("register client: %w", err)
	}

	auth, err := oidc.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     reg.ClientId,
		ClientSecret: reg.ClientSecret,
		StartUrl:     aws.String(startURL),
	})
	if err != nil {
		return nil, fmt.Errorf("start device auth: %w", err)
	}

	if err := openBrowser(*auth.VerificationUriComplete); err != nil {
		fmt.Printf("Open this URL in your browser:\n  %s\n", *auth.VerificationUriComplete)
		fmt.Printf("User code: %s\n\n", *auth.UserCode)
	}

	interval := time.Duration(auth.Interval) * time.Second
	if interval == 0 {
		interval = 5 * time.Second
	}

	for {
		time.Sleep(interval)
		tok, err := oidc.CreateToken(ctx, &ssooidc.CreateTokenInput{
			ClientId:     reg.ClientId,
			ClientSecret: reg.ClientSecret,
			DeviceCode:   auth.DeviceCode,
			GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
		})
		if err != nil {
			// AuthorizationPendingException — keep polling
			continue
		}
		return &Token{
			AccessToken: *tok.AccessToken,
			ExpiresAt:   time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second),
		}, nil
	}
}

type AccountRole struct {
	AccountID   string
	AccountName string
	RoleName    string
}

// ListAccountRoles returns all account+role combos accessible with the given token.
func ListAccountRoles(ctx context.Context, region, accessToken string) ([]AccountRole, error) {
	cfg, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := sso.NewFromConfig(cfg)

	var accounts []AccountRole
	var nextAccountToken *string
	for {
		acctOut, err := client.ListAccounts(ctx, &sso.ListAccountsInput{
			AccessToken: aws.String(accessToken),
			NextToken:   nextAccountToken,
		})
		if err != nil {
			return nil, fmt.Errorf("list accounts: %w", err)
		}
		for _, acct := range acctOut.AccountList {
			var nextRoleToken *string
			for {
				roleOut, err := client.ListAccountRoles(ctx, &sso.ListAccountRolesInput{
					AccessToken: aws.String(accessToken),
					AccountId:   acct.AccountId,
					NextToken:   nextRoleToken,
				})
				if err != nil {
					return nil, fmt.Errorf("list roles for %s: %w", *acct.AccountId, err)
				}
				for _, role := range roleOut.RoleList {
					accounts = append(accounts, AccountRole{
						AccountID:   *acct.AccountId,
						AccountName: *acct.AccountName,
						RoleName:    *role.RoleName,
					})
				}
				if roleOut.NextToken == nil {
					break
				}
				nextRoleToken = roleOut.NextToken
			}
		}
		if acctOut.NextToken == nil {
			break
		}
		nextAccountToken = acctOut.NextToken
	}
	return accounts, nil
}

// OpenBrowser opens the given URL in the default system browser.
func OpenBrowser(url string) error {
	return exec.Command("open", url).Start()
}
