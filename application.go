package geotrigger_golang

type application struct {
	clientId     string
	clientSecret string
	accessToken  string
	expiresIn    int
}

func (application *application) requestAccess(errorChan chan error) {
	return
}

func (application *application) geotriggerAPIRequest(route string, params map[string]interface{},
	responseJSON interface{}, errorChan chan error) {
	return
}

func (application *application) getSessionInfo() map[string]string {
	return map[string]string {
		"access_token": application.accessToken,
		"client_id": application.clientId,
		"client_secret": application.clientSecret,
	}
}

func (application *application) tokenManager() {
	return
}
