package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

type loginResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type validationResult struct {
	Active bool `json:"active"`
}

type AuthConfigResult struct {
	AccessToken  string `yaml:"access_token"`
	RefreshToken string `yaml:"refresh_token"`
	CurrentEnv   string `yaml:"current_environment"` // ignored - used just for compatibility
}

type AuthConfig struct {
	ClientId      string
	RedirectUri   string
	LoginUrl      string
	TokenUrl      string
	ValidationUrl string
	RefreshUrl    string
	Scopes        []string
}

type Auth struct {
	httpClient   *http.Client
	config       AuthConfig
	configFile   string
	accessToken  string
	refreshToken string
	codeVerifier *codeVerifier
}

var authKey contextKey = "magalu.cloud/core/Authentication"

func NewAuthContext(parentCtx context.Context, auth *Auth) context.Context {
	return context.WithValue(parentCtx, authKey, auth)
}
func AuthFromContext(ctx context.Context) *Auth {
	a, _ := ctx.Value(authKey).(*Auth)
	return a
}

func NewAuth(config AuthConfig, client *http.Client) *Auth {
	// For now we are following the IDM convention to allow the users to use IDM
	// when authenticating.
	filePath, err := authFilePath(".mgc.yaml")
	if err != nil {
		return nil
	}

	return &Auth{
		httpClient:   client,
		config:       config,
		configFile:   filePath,
		codeVerifier: nil,
	}
}

/*
Returns the current user access token.

It will first look into the environment variable "MGC_SDK_ACCESS_TOKEN", if
not set, it will try to read from the auth configuration file.

If the token is set as an empty string in either of those it will be treated
as an error
*/
func (o *Auth) AccessToken() (string, error) {
	if o.accessToken != "" {
		return o.accessToken, nil
	}

	if token := os.Getenv("MGC_SDK_ACCESS_TOKEN"); token != "" {
		return token, nil
	}

	authResult, err := o.readConfigFile()
	if err != nil {
		return "", err
	}
	if authResult.AccessToken == "" {
		return "", fmt.Errorf("unable to find an access token, please authenticate yourself first")
	}

	return authResult.AccessToken, nil
}

func (o *Auth) RedirectUri() string {
	return o.config.RedirectUri
}

func (o *Auth) setToken(token *loginResult) error {
	// Always update the tokens, this way the user can assume the Auth object is
	// up-to-date after this function, even in case of a persistance error
	o.accessToken = token.AccessToken
	o.refreshToken = token.RefreshToken

	authResult, err := o.readConfigFile()
	// Ignore if config file doesn't exist
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if authResult == nil {
		authResult = &AuthConfigResult{}
	}

	authResult.AccessToken = token.AccessToken
	authResult.RefreshToken = token.RefreshToken

	err = o.writeConfigFile(authResult)
	if err != nil {
		return err
	}

	return nil
}

func (o *Auth) CodeChallengeToURL() (*url.URL, error) {
	config := o.config
	loginUrl, err := url.Parse(config.LoginUrl)
	if err != nil {
		return nil, err
	}
	codeVerifier, err := newVerifier()
	o.codeVerifier = codeVerifier
	if err != nil {
		return nil, err
	}

	query := loginUrl.Query()
	query.Add("response_type", "code")
	query.Add("client_id", config.ClientId)
	query.Add("redirect_uri", config.RedirectUri)
	query.Add("code_challenge", o.codeVerifier.CodeChallengeS256())
	query.Add("code_challenge_method", "S256")
	query.Add("scope", strings.Join(config.Scopes, " "))

	loginUrl.RawQuery = query.Encode()

	return loginUrl, nil
}

/** Creates a new request access token from authorization code request, be
 * mindful that the code verifier used in this request come from the last call
 * of `CodeChallengeToUrl` method. */
func (o *Auth) RequestAuthTokeWithAuthorizationCode(authCode string) error {
	if o.codeVerifier == nil {
		return fmt.Errorf("no code verification provided, first execute a code challenge request")
	}
	config := o.config
	data := url.Values{}
	data.Set("client_id", config.ClientId)
	data.Set("redirect_uri", config.RedirectUri)
	data.Set("grant_type", "authorization_code")
	data.Set("code", authCode)
	data.Set("code_verifier", o.codeVerifier.value)

	r, err := http.NewRequest(http.MethodPost, config.TokenUrl, strings.NewReader(data.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}

	resp, err := o.httpClient.Do(r)
	if err != nil || resp.StatusCode != http.StatusOK {
		return err
	}

	var result loginResult
	defer resp.Body.Close()
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if err = o.setToken(&result); err != nil {
		return err
	}

	return nil
}

func (o *Auth) ValidateAccessToken() error {
	r, err := o.newValidateAccessTokenRequest()
	if err != nil {
		return err
	}

	resp, err := o.httpClient.Do(r)
	if err != nil || resp.StatusCode != http.StatusOK {
		return err
	}

	var result validationResult
	defer resp.Body.Close()
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if !result.Active {
		return o.RefreshAccessToken()
	}

	return nil
}

func (o *Auth) newValidateAccessTokenRequest() (*http.Request, error) {
	config := o.config
	data := url.Values{}
	data.Set("client_id", config.ClientId)
	data.Set("token_hint", "access_token")
	data.Set("token", o.accessToken)

	r, err := http.NewRequest(http.MethodPost, config.ValidationUrl, strings.NewReader(data.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	return r, err
}

func (o *Auth) RefreshAccessToken() error {
	r, err := o.newRefreshAccessTokenRequest()
	if err != nil {
		return err
	}

	resp, err := o.httpClient.Do(r)
	if err != nil || resp.StatusCode != http.StatusOK {
		return err
	}

	var result loginResult
	defer resp.Body.Close()
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	o.accessToken = result.AccessToken
	o.refreshToken = result.RefreshToken
	return nil
}

func (o *Auth) newRefreshAccessTokenRequest() (*http.Request, error) {
	config := o.config
	data := url.Values{}
	data.Set("client_id", config.ClientId)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", o.refreshToken)

	r, err := http.NewRequest(http.MethodPost, config.RefreshUrl, strings.NewReader(data.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	return r, err
}

func (o *Auth) readConfigFile() (*AuthConfigResult, error) {
	var result AuthConfigResult

	authFile, err := os.ReadFile(o.configFile)
	if err != nil {
		log.Println("unable to read from auth configuration file")
		return nil, err
	}

	err = yaml.Unmarshal(authFile, &result)
	if err != nil {
		log.Println("bad format auth configuration file")
		return nil, err
	}

	return &result, nil
}

func (o *Auth) writeConfigFile(result *AuthConfigResult) error {
	yamlData, err := yaml.Marshal(result)
	if err != nil {
		log.Println("unable to persist auth data")
		return err
	}

	err = os.WriteFile(o.configFile, yamlData, 0600)
	if err != nil {
		return err
	}

	return nil
}

func authFilePath(fName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if homeDir == "" {
		homeDir, err = os.Getwd()
	}
	if homeDir == "" {
		log.Println("unable to locate auth configuration file")
		return "", err
	}

	return path.Join(homeDir, fName), nil
}