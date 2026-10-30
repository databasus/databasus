package systemhttpapi

import "net/http"

type healthResponse struct {
	response GetInstanceHealthResponseObject
}

func (response healthResponse) VisitGetInstanceHealthResponse(writer http.ResponseWriter) error {
	writer.Header().Set("Access-Control-Allow-Origin", "*")
	return response.response.VisitGetInstanceHealthResponse(writer)
}
