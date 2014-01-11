package geotrigger_golang

type Application struct {
	clientId string
	clientSecret string
	accessToken string
	expiresIn int
}

func (application *Application) requestAccess(errorChan chan error) {
	return
}

func (application *Application) geotriggerAPIRequest(route string, params map[string]interface{},
	responseJSON interface{}, errorChan chan error) {
	return
}

func (application *Application) getAccessToken() (string) {
	return application.accessToken
}

func (application *Application) getRefreshToken() (string) {
	return ""
}
