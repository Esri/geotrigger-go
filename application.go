package geotrigger_golang

type application struct {
	clientId     string
	clientSecret string
	accessToken  string
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
		"access_token":  application.accessToken,
		"client_id":     application.clientId,
		"client_secret": application.clientSecret,
	}
}

func newApplication(clientId string, clientSecret string) (Session, chan error) {
	application := &application{
		clientId:     clientId,
		clientSecret: clientSecret,
	}

	return application, sessionInit(application)
}

func (application *application) requestAccess(errorChan chan error) {
	return
}

func (application *application) tokenManager() {
	return
}

func (application *application) request(route string, params map[string]interface{},
	responseJSON interface{}, errorChan chan error) {
	errorChan <- nil
}
