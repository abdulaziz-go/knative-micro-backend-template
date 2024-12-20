package function

import (
	"encoding/json"
	"function/pkg"
	"io"
	"net/http"
	"time"

	sdk "github.com/ucode-io/ucode_sdk"
)

/*
Answer below questions before starting the function.

When the function invoked?
  - table_slug -> AFTER | BEFORE | HTTP -> CREATE | UPDATE | MULTIPLE_UPDATE | DELETE | APPEND_MANY2MANY | DELETE_MANY2MANY

What does it do?
- Explain the purpose of the function.(O'zbekcha yozilsa ham bo'ladi.)
*/

const (
	appId          = ""
	functionName   = ""
	baseUrl        = "https://api.admin.u-code.io"
	projectId      = ""
	requestTimeout = 5 * time.Second
)

func Handler(params *pkg.Params) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			ucodeApi = sdk.New(&sdk.Config{
				BaseURL:        baseUrl,
				FunctionName:   "",
				RequestTimeout: requestTimeout,
				ProjectId:      projectId,
			})

			request  sdk.Request
			response sdk.Response
		)

		ucodeApi.Config().AppId = appId
		{
			requestByte, err := io.ReadAll(r.Body)
			if err != nil {
				handleResponse(w, returnError("error when getting request", err.Error()), http.StatusBadRequest)
				return
			}

			if err = json.Unmarshal(requestByte, &request); err != nil {
				handleResponse(w, returnError("error when unmarshl request", err.Error()), http.StatusInternalServerError)
				return
			}
		}

		params.Log.Info().Msgf("Request: %v", request)

		response.Status = "done"

		handleResponse(w, response, 200)
	}
}

func returnError(clientError string, errorMessage string) interface{} {
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
