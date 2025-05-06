package handlers

import (
	"encoding/json"
	"function/pkg"
	"net/http"

	cache "github.com/golanguzb70/redis-cache"
	"github.com/rs/zerolog"
	sdk "github.com/ucode-io/ucode_sdk"
)

type Handler struct {
	Log        zerolog.Logger
	params     *pkg.Params
	ucodeApi   sdk.UcodeApis
	redisCache cache.RedisCache
}

func NewHandler(params *pkg.Params) Handler {
	return Handler{
		params:     params,
		Log:        params.Log,
		ucodeApi:   params.UcodeApi,
		redisCache: params.CacheClient,
	}
}

func (h Handler) returnError(clientError string, errorMessage string) interface{} {
	h.Log.Error().Msgf("Error: %s, ErrorMessage: %s", clientError, errorMessage)
	return sdk.Response{
		Status: "error",
		Data:   map[string]interface{}{"message": clientError, "error": errorMessage},
	}
}

func handleResponse(w http.ResponseWriter, body interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")

	bodyByte, err := json.Marshal(body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`
			{
				"error": "Error marshalling response"
			}
		`))
		return
	}

	w.WriteHeader(statusCode)
	w.Write(bodyByte)
}
