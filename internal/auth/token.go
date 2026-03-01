// Package auth provides Azure AD token acquisition for DTS API calls.
package auth

import (
	"context"
	"fmt"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// DTS resource scope - server-side RBAC determines actual permissions.
const dtsScope = "https://durabletask.io/.default"

// TokenProvider acquires and caches Azure AD bearer tokens for DTS API calls.
type TokenProvider struct {
	cred     azcore.TokenCredential
	tenantID string
	mu       sync.Mutex
}

// NewTokenProvider creates a token provider using the specified auth mode.
// Supported modes: "default", "browser", "cli", "device", "none".
// "none" returns nil (no auth) for use with the DTS emulator.
func NewTokenProvider(mode string, tenantID string) (*TokenProvider, error) {
	if mode == "none" {
		return nil, nil
	}

	tp := &TokenProvider{tenantID: tenantID}

	var err error
	switch mode {
	case "browser":
		opts := &azidentity.InteractiveBrowserCredentialOptions{}
		if tenantID != "" {
			opts.TenantID = tenantID
		}
		tp.cred, err = azidentity.NewInteractiveBrowserCredential(opts)
	case "cli":
		opts := &azidentity.AzureCLICredentialOptions{}
		if tenantID != "" {
			opts.TenantID = tenantID
		}
		tp.cred, err = azidentity.NewAzureCLICredential(opts)
	case "device":
		opts := &azidentity.DeviceCodeCredentialOptions{}
		if tenantID != "" {
			opts.TenantID = tenantID
		}
		tp.cred, err = azidentity.NewDeviceCodeCredential(opts)
	case "default", "":
		opts := &azidentity.DefaultAzureCredentialOptions{}
		if tenantID != "" {
			opts.TenantID = tenantID
		}
		tp.cred, err = azidentity.NewDefaultAzureCredential(opts)
	default:
		return nil, fmt.Errorf("unsupported auth mode: %q (use default, browser, cli, device, or none)", mode)
	}

	if err != nil {
		return nil, fmt.Errorf("creating %s credential: %w", mode, err)
	}
	return tp, nil
}

// GetToken acquires a bearer token for the DTS resource.
// azidentity handles caching internally.
func (tp *TokenProvider) GetToken(ctx context.Context) (string, error) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	token, err := tp.cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{dtsScope},
	})
	if err != nil {
		return "", fmt.Errorf("acquiring token: %w", err)
	}
	return token.Token, nil
}
