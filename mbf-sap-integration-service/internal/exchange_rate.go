package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"function/pkg"

	"io"
	"net/http"
	"time"
)

func (h *Handler) CreateExchangeRate() error {
	rates, err := getExchangeRate()
	if err != nil {
		fmt.Println("Error on getting exchange rate:", err)
		return err
	}

	var uzs, ruble, eur, cny float64

	for _, rate := range rates {
		currency, ok := rate["Currency"].(string)
		if !ok {
			fmt.Println("Invalid currency format")
			continue
		}

		rateValue, ok := rate["Rate"].(float64)
		if !ok {
			fmt.Println("Invalid rate format for currency:", currency)
			continue
		}

		switch currency {
		case "UZS":
			uzs = rateValue
		case "руб":
			ruble = rateValue
		case "EUR":
			eur = rateValue
		case "CNY":
			cny = rateValue
		default:
			fmt.Println("Unrecognized currency:", currency)
		}
	}

	fmt.Println(">>>>>>", uzs, ruble, eur, cny)

	var (
		createModelCat    = pkg.SingleURL + "exchange_rate"
		createModelCatReq = pkg.Request{
			Data: map[string]interface{}{
				"ruble": ruble,
				"uzs":   uzs,
				"eur":   eur,
				"cny":   cny,
				"date":  time.Now().Format(time.RFC3339),
			},
		}
	)

	_, err = pkg.DoRequest(createModelCat, "POST", createModelCatReq)
	if err != nil {
		fmt.Println("Error on creating modelCat:", err)
		return err
	}
	return nil
}

func getExchangeRate() ([]map[string]interface{}, error) {
	var (
		url    = "https://212.83.166.117:50000/b1s/v1/SQLQueries('getExchangeRateWithDate')/List"
		method = "GET"
		rates  []map[string]interface{}
		body   = map[string]interface{}{
			"ParamList": fmt.Sprintf("currDate='%s'", time.Now().Format("20060102")),
		}
	)

	reqBody, err := json.Marshal(body)
	if err != nil {
		return rates, err
	}

	var rate pkg.SAPB1Response

	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Println("Request creation error:", err)
		return rates, err
	}

	// req.Header.Add("Content-Type", "application/json")
	req.Header.Add("SessionId", pkg.SessionId)
	req.Header.Add("Cookie", fmt.Sprintf("B1SESSION=%s; ROUTEID=.node4", pkg.SessionId))

	res, err := pkg.Client.Do(req)
	if err != nil {
		fmt.Println("Request error:", err)
		return rates, err
	}
	defer res.Body.Close()

	resByte, err := io.ReadAll(res.Body)
	if err != nil {
		return rates, err
	}

	if err := json.Unmarshal(resByte, &rate); err != nil {
		fmt.Println("Unmarshal error:", err)
		return rates, err
	}

	rates = append(rates, rate.Value...)

	fmt.Println("RESPONSE BODY: ", string(resByte))

	return rates, nil
}
