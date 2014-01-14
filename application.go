package geotrigger_golang

type application struct {
	TokenManager
	clientId     string
	clientSecret string
	expiresIn    int
}

func (application *application) Request(route string, params map[string]interface{},
	responseJSON interface{}) chan error {
	errorChan := make(chan error)
	go application.request(route, params, responseJSON, errorChan)
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
	// TODO: set up token manager and access token here
	return
}

func (application *application) tokenManager() {
	return
}

func (application *application) request(route string, params map[string]interface{},
	responseJSON interface{}, errorChan chan error) {
	errorChan <- nil
}
