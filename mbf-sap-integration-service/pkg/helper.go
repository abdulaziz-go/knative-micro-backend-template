package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	sdk "github.com/ucode-io/ucode_sdk"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func CreateQueryForTable() error {
	var (
		payload = strings.NewReader(`{
			"SqlCode": "OINVDEPGet",
			"SqlName": "GetOINVDEP",
			"SqlText": "SELECT T0.Code, T0.Name,T0.U_finance FROM [@OINVDEP] T0"
		}`)

		url    = "https://212.83.166.117:50000/b1s/v1/SQLQueries"
		method = "POST"
	)

	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		fmt.Println("Request creation error:", err)
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("SessionId", SessionId)

	res, err := Client.Do(req)
	if err != nil {
		fmt.Println("Request error:", err)
		return err
	}
	defer res.Body.Close()

	return nil
}

func LoginSAP() error {
	var (
		loginSAPResponse LoginSAPResponse

		url    = "https://212.83.166.117:50000/b1s/v1/Login"
		method = "POST"

		payload = strings.NewReader(`{
    		"CompanyDB": "MBF_INTEGRATION_TEST", 
    		"Password": "q1w2e3r4T%",
    		"UserName": "manager"
		}`)
	)

	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		fmt.Println("Request creation error:", err)
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	res, err := Client.Do(req)
	if err != nil {
		fmt.Println("Request error:", err)
		return err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, &loginSAPResponse); err != nil {
		fmt.Println("Unmarshal error:", err)
		return err
	}

	SessionId = loginSAPResponse.SessionId

	return nil
}

func HandleResponse(w http.ResponseWriter, body interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")

	bodyByte, err := json.Marshal(body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(bodyByte)
		return
	}

	w.WriteHeader(statusCode)
	w.Write(bodyByte)
}

func DoRequest(url string, method string, body interface{}) ([]byte, error) {
	data, err := json.Marshal(&body)
	if err != nil {
		return nil, nil
	}
	client := &http.Client{
		Timeout: time.Duration(60 * time.Second),
	}

	request, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, nil
	}

	request.Header.Add("authorization", "API-KEY")

	resp, err := client.Do(request)
	if err != nil {
		return nil, nil
	}
	defer resp.Body.Close()

	respByte, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil
	}

	return respByte, nil
}

func GetStringValue(data map[string]interface{}, key string) string {
	if value, ok := data[key].(string); ok {
		return value
	}
	return ""
}

func GetIntValue(data map[string]interface{}, key string) int {
	if value, ok := data[key].(int); ok {
		return value
	}
	return 0
}

func MongoConn() (*mongo.Database, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(MongoURL))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
		return nil, err
	}
	fmt.Println("Connected to MongoDB!")

	return client.Database("mbf_ebf657726f964d5fb08c65c915f85e2c_p_obj_build_svcs"), err
}

func ReturnError(clientError string, errorMessage string) interface{} {
	return sdk.Response{
		Status: "error",
		Data:   map[string]interface{}{"message": clientError, "error": errorMessage},
	}
}
