package googledrive

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

// accessToken resolves the best available credential source and returns an OAuth access token plus its credential type.
func (c connector) accessToken(ctx context.Context, metadata map[string]string) (string, string, error) {
	if token := strings.TrimSpace(metadata[MetadataAccessToken]); token != "" {
		return token, "access_token", nil
	}
	if token := strings.TrimSpace(os.Getenv(googleDriveAccessTokenEnv)); token != "" {
		return token, "access_token", nil
	}

	if serviceAccountPath := firstNonEmpty(metadata[MetadataServiceAccountPath], os.Getenv(googleDriveServiceAccountEnv)); serviceAccountPath != "" {
		token, err := c.serviceAccountToken(ctx, serviceAccountPath)
		return token, "service_account", err
	}

	if credentialsPath := firstNonEmpty(metadata[MetadataOAuthCredentialsPath], os.Getenv(googleDriveOAuthCredentialsEnv)); credentialsPath != "" {
		token, err := c.authorizedUserToken(ctx, credentialsPath)
		return token, "oauth", err
	}

	return "", "", errors.New("google drive credentials are not configured")
}

// authorizedUserToken exchanges an authorized-user refresh token credentials file for an access token.
func (c connector) authorizedUserToken(ctx context.Context, credentialsPath string) (string, error) {
	body, err := os.ReadFile(credentialsPath)
	if err != nil {
		return "", fmt.Errorf("read oauth credentials: %w", err)
	}

	var credentials authorizedUserCredentials
	if err := json.Unmarshal(body, &credentials); err != nil {
		return "", fmt.Errorf("decode oauth credentials: %w", err)
	}
	if credentials.Type != "authorized_user" {
		return "", errors.New("oauth credentials must be an authorized_user JSON file")
	}
	if credentials.ClientID == "" || credentials.ClientSecret == "" || credentials.RefreshToken == "" {
		return "", errors.New("oauth credentials require client_id, client_secret, and refresh_token")
	}

	form := url.Values{}
	form.Set("client_id", credentials.ClientID)
	form.Set("client_secret", credentials.ClientSecret)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", credentials.RefreshToken)
	bodyBytes, _, err := c.postForm(ctx, firstNonEmpty(credentials.TokenURI, defaultGoogleTokenURL), form.Encode())
	if err != nil {
		return "", err
	}

	var token tokenResponse
	if err := json.Unmarshal(bodyBytes, &token); err != nil {
		return "", fmt.Errorf("decode oauth token response: %w", err)
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return "", errors.New("oauth token response missing access_token")
	}
	return token.AccessToken, nil
}

// serviceAccountToken signs a service-account JWT assertion and exchanges it for an access token.
func (c connector) serviceAccountToken(ctx context.Context, credentialsPath string) (string, error) {
	body, err := os.ReadFile(credentialsPath)
	if err != nil {
		return "", fmt.Errorf("read service account credentials: %w", err)
	}

	var credentials serviceAccountCredentials
	if err := json.Unmarshal(body, &credentials); err != nil {
		return "", fmt.Errorf("decode service account credentials: %w", err)
	}
	if credentials.Type != "service_account" {
		return "", errors.New("service account credentials must be a service_account JSON file")
	}
	if credentials.ClientEmail == "" || credentials.PrivateKey == "" {
		return "", errors.New("service account credentials require client_email and private_key")
	}

	assertion, err := signedJWT(credentials)
	if err != nil {
		return "", err
	}

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", assertion)
	bodyBytes, _, err := c.postForm(ctx, firstNonEmpty(credentials.TokenURI, defaultGoogleTokenURL), form.Encode())
	if err != nil {
		return "", err
	}

	var token tokenResponse
	if err := json.Unmarshal(bodyBytes, &token); err != nil {
		return "", fmt.Errorf("decode service account token response: %w", err)
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return "", errors.New("service account token response missing access_token")
	}
	return token.AccessToken, nil
}

// signedJWT creates a short-lived RS256 JWT assertion for Google service-account OAuth.
func signedJWT(credentials serviceAccountCredentials) (string, error) {
	block, _ := pem.Decode([]byte(credentials.PrivateKey))
	if block == nil {
		return "", errors.New("decode service account private key: missing pem block")
	}

	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse service account private key: %w", err)
	}
	privateKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return "", errors.New("service account private key must be RSA")
	}

	now := time.Now().UTC()
	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	claims := map[string]any{
		"iss":   credentials.ClientEmail,
		"scope": defaultScope,
		"aud":   firstNonEmpty(credentials.TokenURI, defaultGoogleTokenURL),
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal jwt header: %w", err)
	}
	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal jwt claims: %w", err)
	}

	signingInput := base64.RawURLEncoding.EncodeToString(headerBytes) + "." + base64.RawURLEncoding.EncodeToString(claimsBytes)
	hash := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("sign jwt assertion: %w", err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}
