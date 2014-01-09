package geotrigger_golang

type Application struct {
	ClientId string
	ClientSecret string
	AccessToken string
	ExpiresIn int
}

func (application *Application) RequestAccess() (err error) {
	return
}

func (application *Application) GeotriggerAPIRequest(route string, data map[string]interface{}, responseJSON interface{}) (error) {
	return nil
}
