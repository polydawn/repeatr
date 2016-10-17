package gs

import (
	"fmt"
	"io/ioutil"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/storage/v1"

	"go.polydawn.net/repeatr/api/def"
)

/*
 Environment variable used to directly pass an OAuth2 access token to repeatr
*/
const EnvGsAccessToken = "GS_ACCESS_TOKEN"

/*
 Environment variable used to pass the path of
 a Google Developers service account JSON key file
 which is then used for two-legged OAuth to create an Access Token
*/
const EnvGsAccountFile = "GS_SERVICE_ACCOUNT_FILE"

func GetAccessToken() (*oauth2.Token, error) {
	var err error
	var secrets []byte
	token := getAccessTokenFromEnv()
	if token != nil {
		return token, nil
	}
	secrets, err = getServiceAccountSecretsFromEnv()
	if err == nil {
		token, err = createAccessTokenFromSecrets(secrets, []string{storage.DevstorageReadWriteScope})
	}
	return token, err
}

func mustLoadToken() *oauth2.Token {
	token, err := GetAccessToken()
	if err == nil {
		return token
	}
	panic(&def.ErrConfigValidation{
		Msg: "gs credentials missing.  set GS_ACCESS_TOKEN or GS_SERVICE_ACCOUNT_FILE.",
	})
}

func getAccessTokenFromEnv() *oauth2.Token {
	accessToken := os.Getenv(EnvGsAccessToken)
	if accessToken != "" {
		return &oauth2.Token{AccessToken: accessToken}
	}
	return nil
}

func getServiceAccountSecretsFromEnv() ([]byte, error) {
	secretPath := os.Getenv(EnvGsAccountFile)
	if secretPath == "" {
		return nil, fmt.Errorf("Environment variable %s not set", EnvGsAccountFile)
	}
	serviceAccountSecrets, err := ioutil.ReadFile(secretPath)
	return serviceAccountSecrets, err
}

func createAccessTokenFromSecrets(authSecretsJson []byte, scopes []string) (*oauth2.Token, error) {
	// This will require the following fields from the Google Developrs service account JSON:
	//  "client_email"
	//  "private_key"
	config, err := google.JWTConfigFromJSON(authSecretsJson, scopes...)
	if err != nil {
		return nil, err
	}

	token, err := config.TokenSource(oauth2.NoContext).Token()
	if err != nil {
		return nil, err
	}

	return token, nil
}
