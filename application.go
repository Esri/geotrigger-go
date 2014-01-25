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

func (application *application) Request(route string, params map[string]interface{},
	responseJSON interface{}) chan error {
	errorChan := make(chan error)
	go func() {
		errorChan <- geotriggerPost(application, route, params, responseJSON)
	}()
	return errorChan
}

func (application *application) GetSessionInfo() map[string]string {
	return map[string]string{
		"access_token":  application.getAccessToken(),
		"client_id":     application.clientId,
		"client_secret": application.clientSecret,
	}
}

func newApplication(clientId string, clientSecret string) (Session, chan error) {
	application := &application{
		clientId:     clientId,
		clientSecret: clientSecret,
	}

	errorChan := make(chan error)
	go application.requestAccess(errorChan)

	return application, errorChan
}

func (application *application) requestAccess(errorChan chan error) {
	var appTokenResponse applicationTokenResponse
	if err := agoPost(ago_token_route, application.prepareTokenRequestValues(), &appTokenResponse); err != nil {
		go func() {
			errorChan <- err
		}()
	}

	// store the new access token
	application.tokenManager = newTokenManager(appTokenResponse.AccessToken, "", appTokenResponse.ExpiresIn)

	go func() {
		errorChan <- nil
	}()
	return
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
