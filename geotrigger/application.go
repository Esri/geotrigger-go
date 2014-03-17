package geotrigger

import (
	"net/url"
)

type application struct {
	tokenManager
	clientID     string
	clientSecret string
	env          *environment
}

type applicationTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

func (application *application) request(route string, params map[string]interface{}, responseJSON interface{}) error {
	return geotriggerPost(application.env, application, route, params, responseJSON)
}

func (application *application) info() map[string]string {
	return map[string]string{
		"access_token":  application.getAccessToken(),
		"client_id":     application.clientID,
		"client_secret": application.clientSecret,
	}
}

func newApplication(clientID string, clientSecret string) (session, error) {
	application := &application{
		clientID:     clientID,
		clientSecret: clientSecret,
		env:          defEnv,
	}
	return application, application.requestAccess()
}

func (application *application) requestAccess() error {
	var appTokenResponse applicationTokenResponse
	if err := agoPost(application.env, ago_token_route, application.prepareTokenRequestValues(),
		&appTokenResponse); err != nil {
		return err
	}

	// store the new access token
	application.tokenManager = newTokenManager(appTokenResponse.AccessToken, "", appTokenResponse.ExpiresIn)

	return nil
}

func (application *application) refresh(refreshToken string) error {
	var appTokenResponse applicationTokenResponse
	if err := agoPost(application.env, ago_token_route, application.prepareTokenRequestValues(),
		&appTokenResponse); err != nil {
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
	values.Set("client_id", application.clientID)
	values.Set("client_secret", application.clientSecret)
	values.Set("grant_type", "client_credentials")
	values.Set("f", "json")
	return []byte(values.Encode())
}

func (application *application) setEnv(env *environment) {
	application.env = env
}
