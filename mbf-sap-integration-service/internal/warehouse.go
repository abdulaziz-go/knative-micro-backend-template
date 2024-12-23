package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"function/pkg"
	"io"
	"net/http"
)

func (h *Handler)CreateOrUpdateWarehouse(warehouses []map[string]interface{}) error {

	for _, wh := range warehouses {
		fmt.Println(wh)
		fmt.Println("creating...")
		var (
			createWHURL = pkg.SingleURL + "warehouse"
			createWHReq = pkg.Request{
				Data: map[string]interface{}{
					"name": wh["WarehouseName"],
					"code": wh["WarehouseCode"],
				},
			}
		)

		_, err := pkg.DoRequest(createWHURL, "POST", createWHReq)
		if err != nil {
			fmt.Println("Error on creating product:", err)
			return err
		}
	}

	return nil
}

func GetWarehouse() ([]map[string]interface{}, error) {
	var (
		pagination = "Warehouses?$select=WarehouseName,WarehouseCode"
		url        = "https://212.83.166.117:50000/b1s/v1/"
		method     = "GET"

		warehouses []map[string]interface{}
	)

	for {
		var warehouse pkg.SAPB1Response

		req, err := http.NewRequest(method, url+pagination, nil)
		if err != nil {
			fmt.Println("Request creation error:", err)
			return warehouses, err
		}

		// req.Header.Add("Content-Type", "application/json")
		req.Header.Add("SessionId", pkg.SessionId)
		req.Header.Add("Cookie", fmt.Sprintf("B1SESSION=%s; ROUTEID=.node4", pkg.SessionId))

		res, err := pkg.Client.Do(req)
		if err != nil {
			fmt.Println("Request error:", err)
			return warehouses, err
		}
		defer res.Body.Close()

		resByte, err := io.ReadAll(res.Body)
		if err != nil {
			return warehouses, err
		}

		if err := json.Unmarshal(resByte, &warehouse); err != nil {
			fmt.Println("Unmarshal error:", err)
			return warehouses, err
		}

		warehouses = append(warehouses, warehouse.Value...)

		if warehouse.OdataNextLink == "" {
			break
		}

		pagination = warehouse.OdataNextLink
		fmt.Println("PAGE", pagination)
		// fmt.Println("RESPONSE BODY: ", string(resByte))
	}

	return warehouses, nil
}

// if in ucode platform warehouse created
// then create that also in SAP
// then update warehouse code in ucode db
func CreateWarehouse(guid, warehouseName string) error {

	var (
		url     = "https://212.83.166.117:50000/b1s/v1/Warehouses"
		reqBody = map[string]interface{}{
			"WarehouseCode": "",
			"WarehouseName": "",
		}
	)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return err
	}

	_, err = http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Request creation error warehouse:", err)
		return err
	}

	return nil
}
