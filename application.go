package geotrigger_golang

type Application struct {
	ClientId string
	ClientSecret string
	AccessToken string
	ExpiresIn int32
}

func (application *Application) RequestAccess() (err error) {
	return
}

func (application *Application) GeotriggerAPIRequest(route string, data map[string]interface{}, jsonContainer interface{}) (error) {
	return nil
}
