package geotrigger_golang

import (
	"net/url"
)

type application struct {
	tokenManager
	clientId     string
	clientSecret string
}

type applicationTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

func (application *application) request(route string, params map[string]interface{}, responseJSON interface{}) error {
	return geotriggerPost(application, route, params, responseJSON)
}

func (application *application) info() map[string]string {
	return map[string]string{
		"access_token":  application.getAccessToken(),
		"client_id":     application.clientId,
		"client_secret": application.clientSecret,
	}
}

func newApplication(clientId string, clientSecret string) (session, error) {
	application := &application{
		clientId:     clientId,
		clientSecret: clientSecret,
	}

	if err := application.requestAccess(); err != nil {
		return nil, err
	}

	return application, nil
}

func (application *application) requestAccess() error {
	var appTokenResponse applicationTokenResponse
	if err := agoPost(ago_token_route, application.prepareTokenRequestValues(), &appTokenResponse); err != nil {
		return err
	}

	// store the new access token
	application.tokenManager = newTokenManager(appTokenResponse.AccessToken, "", appTokenResponse.ExpiresIn)

	return nil
}

func (application *application) refresh(refreshToken string) error {
	var appTokenResponse applicationTokenResponse
	if err := agoPost(ago_token_route, application.prepareTokenRequestValues(), &appTokenResponse); err != nil {
		return err
	}

	// store the new access token
	application.setExpiresAt(appTokenResponse.ExpiresIn)
	application.setAccessToken(appTokenResponse.AccessToken)

	return nil
}

func (application *application) prepareTokenRequestValues() []byte {
	// prep values
	values := url.Values{}
	values.Set("client_id", application.clientId)
	values.Set("client_secret", application.clientSecret)
	values.Set("grant_type", "client_credentials")
	values.Set("f", "json")
	return []byte(values.Encode())
}
