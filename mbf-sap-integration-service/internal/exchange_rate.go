// 1. Login to SAP
// 2. check if in ucode today's exchange rate is already exist if yes update else create new one with sap data
// Note: every action in mobile should be after creating exchange rate
package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"function/pkg"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	exchangeRateURL = "https://212.83.166.117:50000/b1s/v1/SQLQueries('getExchangeRateWithDate')/List"
	dateFormat      = "20060102"
	nodeRouteID     = ".node4"
)

type exchangeRate struct {
	ruble float64
	uzs   float64
	eur   float64
	cny   float64
}

func (h *Handler) ExchangeRate() error {
	// Step 1: Login to SAP
	if err := pkg.LoginSAP(); err != nil {
		h.Log.Err(err).Msg("Failed to login to SAP")
		return err
	}

	// Step 2: Parse the exchange rate data from SAP
	exchangeRate, err := fetchAndParseExchangeRate()
	if err != nil {
		h.Log.Err(err).Msg("Failed to fetch and parse exchange rate")
		return err
	}

	// Step 3: Send the data to the local service
	if err := h.updateOrCreateExchangeRate(exchangeRate); err != nil {
		h.Log.Err(err).Msg("Failed to update or create exchange rate")
		return err
	}

	return nil
}

func (h *Handler) updateOrCreateExchangeRate(exchangeRate exchangeRate) error {
	var (
		collection = h.MongoDB.Collection("exchange_rates")

		now        = time.Now().UTC()
		startOfDay = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		endOfDay   = startOfDay.Add(24 * time.Hour)

		filter = bson.M{
			"createdAt": bson.M{
				"$gte": startOfDay,
				"$lt":  endOfDay,
			},
		}
		update = bson.M{
			"$set": bson.M{

				"ruble":     exchangeRate.ruble,
				"uzs":       exchangeRate.uzs,
				"eur":       exchangeRate.eur,
				"cny":       exchangeRate.cny,
				"date":      time.Now().Format(time.RFC3339),
				"updatedAt": time.Now(),
			},
		}
	)
	result, err := collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		h.Log.Err(err).Msg("Error on counting exchange rate")
		return err
	}

	if result.MatchedCount == 0 {
		h.Log.Info().Msg("Creating new exchange rate")
		_, err := collection.InsertOne(context.TODO(), bson.M{
			"__v":       0,
			"guid":      uuid.New().String(),
			"ruble":     exchangeRate.ruble,
			"uzs":       exchangeRate.uzs,
			"eur":       exchangeRate.eur,
			"cny":       exchangeRate.cny,
			"date":      time.Now().Format(time.RFC3339),
			"createdAt": time.Now(),
			"updatedAt": time.Now(),
		})
		if err != nil {
			h.Log.Err(err).Msg("Error on creating exchange rate")
			return err
		}
	}

	return nil

}

// fetchAndParseExchangeRate retrieves and parses the exchange rates from SAP
func fetchAndParseExchangeRate() (exchangeRate, error) {
	sapRates, err := fetchSAPExchangeRate()
	if err != nil {
		return exchangeRate{}, fmt.Errorf("error fetching SAP exchange rate: %w", err)
	}

	return parseExchangeRate(sapRates)
}

// fetchSAPExchangeRate retrieves exchange rate data from SAP
func fetchSAPExchangeRate() ([]map[string]interface{}, error) {
	body := map[string]interface{}{
		"ParamList": fmt.Sprintf("currDate='%s'", time.Now().Format(dateFormat)),
	}
	reqBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("GET", exchangeRateURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Add("SessionId", pkg.SessionId)
	req.Header.Add("Cookie", fmt.Sprintf("B1SESSION=%s; ROUTEID=%s", pkg.SessionId, nodeRouteID))

	res, err := pkg.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request error: %w", err)
	}
	defer res.Body.Close()

	resBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var sapResponse pkg.SAPB1Response
	if err := json.Unmarshal(resBytes, &sapResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SAP response: %w", err)
	}

	return sapResponse.Value, nil
}

// parseExchangeRate maps the SAP response to the ExchangeRate struct
func parseExchangeRate(sapRates []map[string]interface{}) (exchangeRate, error) {
	var resultRate = exchangeRate{}
	for _, rate := range sapRates {
		currency, ok := rate["Currency"].(string)
		if !ok {
			continue
		}

		rateValue, ok := rate["Rate"].(float64)
		if !ok {
			continue
		}

		switch currency {
		case "руб":
			resultRate.ruble = rateValue
		case "UZS":
			resultRate.uzs = rateValue
		case "EUR":
			resultRate.eur = rateValue

		case "CNY":
			resultRate.cny = rateValue
		default:
			continue

		}

	}

	return resultRate, nil
}
