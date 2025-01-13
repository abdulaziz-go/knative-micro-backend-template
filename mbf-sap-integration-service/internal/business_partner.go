package internal

import (
	"encoding/json"
	"fmt"
	"function/pkg"

	"io"
	"net/http"
)

func (h *Handler)CreateBusinessPartner(businessPartners []map[string]interface{}) error {
	for _, data := range businessPartners {
		var groupCode string

		switch data["GroupCode"].(int) {
		case 100:
			groupCode = "client"
		case 101:
			groupCode = "supplier"
		case 107:
			groupCode = "employee"
		default:
			continue
		}

		var (
			url     = pkg.SingleURL + "account"
			reqBody = pkg.Request{
				Data: map[string]interface{}{
					"phone_number":    data["Phone1"],
					"group_code":      []string{groupCode},
					"current_balance": data["CurrentAccountBalance"],
					"code":            data["CardCode"],
					"name":            data["CardName"],
				},
			}
		)

		_, err := pkg.DoRequest(url, "POST", reqBody)
		if err != nil {
			fmt.Println("Error on creating modelCat:", err)
			return err
		}
	}
	return nil
}

func (h *Handler)GetBusinessPartner() ([]map[string]interface{}, error) {
	var (
		pagination       = "BusinessPartners?$select=CardCode,CardName,GroupCode,Phone1"
		url              = "https://212.83.166.117:50000/b1s/v1/"
		method           = "GET"
		businessPartners []map[string]interface{}
	)
	for {

		var businessPartner pkg.SAPB1Response

		req, err := http.NewRequest(method, url+pagination, nil)
		if err != nil {
			fmt.Println("Request creation error:", err)
			return businessPartners, err
		}

		// req.Header.Add("Content-Type", "application/json")
		req.Header.Add("SessionId", pkg.SessionId)
		req.Header.Add("Cookie", fmt.Sprintf("B1SESSION=%s; ROUTEID=.node4", pkg.SessionId))

		res, err := pkg.Client.Do(req)
		if err != nil {
			fmt.Println("Request error:", err)
			return businessPartners, err
		}
		defer res.Body.Close()

		resByte, err := io.ReadAll(res.Body)
		if err != nil {
			return businessPartners, err
		}

		if err := json.Unmarshal(resByte, &businessPartner); err != nil {
			fmt.Println("Unmarshal error:", err)
			return businessPartners, err
		}

		businessPartners = append(businessPartners, businessPartner.Value...)
		if businessPartner.OdataNextLink == "" {
			break
		}

		pagination = businessPartner.OdataNextLink
		fmt.Println("RESPONSE BODY: ", string(resByte))
	}

	return businessPartners, nil
}
