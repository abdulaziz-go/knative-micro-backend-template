package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"function/pkg"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
)

// CreateStock function
// ucode needs api that update from custom query
// because stock will update from custom given itemCode and WarehouseCode
var (
	Warehouses         = map[string]string{}
	ProductAndServices = map[string]string{}
)

func ReadProductAndServices() {
	file, err := os.Open("product_and_service.json")
	if err != nil {
		panic(err)
	}

	byte, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	type Item struct {
		Guid string `json:"guid"`
		Code string `json:"code"`
	}
	var items []Item

	if err := json.Unmarshal(byte, &items); err != nil {
		fmt.Println("error while unmarshalling: ", err)
	}

	for _, item := range items {
		ProductAndServices[item.Code] = item.Guid
	}

}

func ReadWarehouses() {
	file, err := os.Open("warehouses.json")
	if err != nil {
		panic(err)
	}

	byte, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	type Item struct {
		Guid string `json:"guid"`
		Code string `json:"code"`
	}
	var items []Item

	if err := json.Unmarshal(byte, &items); err != nil {
		fmt.Println("error while unmarshalling: ", err)
	}

	for _, item := range items {
		Warehouses[item.Code] = item.Guid
	}
}

func CreateStock(stocks []map[string]interface{}) error {
	ReadProductAndServices()
	ReadWarehouses()

	for _, stock := range stocks {

		var (
			url     = pkg.SingleURL + "stock"
			reqBody = pkg.Request{
				Data: map[string]interface{}{
					"product_and_service_id": ProductAndServices[stock["ItemCode"].(string)],
					"warehouse_id":           Warehouses[stock["WhsCode"].(string)],
					"price":                  stock["AvgPrice"],
					"quantity":               stock["OnHand"],
					"product_code":           stock["ItemCode"],
					// "tannarx":                "",
				},
			}
		)

		_, err := pkg.DoRequest(url, "POST", reqBody)
		if err != nil {
			fmt.Println("Error on creating productAndService:", err)
			continue
		}
		// os.Exit(1)
	}

	return nil
}

func (h *Handler)GetStockV2(collection *mongo.Collection) ([]map[string]interface{}, error) {
	var (
		pagination = "SQLQueries('OITWGETfull')/List"
		url        = "https://212.83.166.117:50000/b1s/v1/"
		method     = "GET"

		stocks []map[string]interface{}
	)

	counter := 0

	for {
		var stock pkg.SAPB1Response

		req, err := http.NewRequest(method, url+pagination, nil)
		if err != nil {
			fmt.Println("Request creation error:", err)
			return stocks, err
		}

		// req.Header.Add("Content-Type", "application/json")
		req.Header.Add("SessionId", pkg.SessionId)
		req.Header.Add("Cookie", fmt.Sprintf("B1SESSION=%s; ROUTEID=.node4", pkg.SessionId))

		res, err := pkg.Client.Do(req)
		if err != nil {
			fmt.Println("Request error:", err)
			return stocks, err
		}
		defer res.Body.Close()

		resByte, err := io.ReadAll(res.Body)
		if err != nil {
			return stocks, err
		}

		if err := json.Unmarshal(resByte, &stock); err != nil {
			fmt.Println("Unmarshal error:", err)
			return stocks, err
		}

		stocks = append(stocks, stock.Value...)

		if stock.OdataNextLink == "" {
			break
		}

		pagination = stock.OdataNextLink
		fmt.Println("PAGE", pagination)
		fmt.Println(counter)
		if counter == 5 {
			bunchInsert(collection, stocks)
			stocks = make([]map[string]interface{}, 0)
			counter = 0
		}
		counter++

	}

	return stocks, nil
}

func bunchInsert(collection *mongo.Collection, sapStock []map[string]interface{}) {
	var mps = []interface{}{}

	for index, stock := range sapStock {
		fmt.Println("INDEX: ", index)
		customDate, _ := time.Parse(time.RFC3339, "2024-11-11T14:02:21.991Z")

		var mp = map[string]interface{}{
			"guid":                   uuid.New().String(),
			"quantity":               stock["OnHand"],
			"price":                  stock["AvgPrice"],
			"product_code":           stock["ItemCode"],
			"warehouse_id":           Warehouses[stock["WhsCode"].(string)],
			"product_and_service_id": ProductAndServices[stock["ItemCode"].(string)],
			"updatedAt":              customDate,
			"createdAt":              customDate,

			// "subdivision_id":         "5c7c0dba-a73b-40bf-b4be-36ef6e58898e",
			// "direction_id":           "5acde960-a08a-4e45-9a95-d38e1c8bf4e6",
			// "category_model_id":      "0ce1a7d0-9eac-42a5-a686-b63a565c0e86",
			// "product_name":           "Рама таёрлов булим бошлиги 26 горный сагадиска",
			// "product_id":             "a1522f93-d543-4a7b-96bc-1e772877df39",
			// "tannarx":                12321,
			// "product_code":      "123",
			// "price":             1200,
			// "product_and_service_id": "e1366747-0231-4aa5-b48c-a9612fbb0659",
			// "warehouse_id":           "cfeae539-aa58-4be2-908a-8d30f29f7398",
			// "quantity":          123,
		}

		mps = append(mps, mp)

	}
	_, err := collection.InsertMany(context.Background(), mps)
	if err != nil {
		fmt.Println("error while inserting: ", err)

	}
}

func GetStock() ([]map[string]interface{}, error) {
	var (
		pagination = "SQLQueries('OITWGETWithstockCount')/List"
		url        = "https://212.83.166.117:50000/b1s/v1/"
		method     = "GET"

		stocks []map[string]interface{}
	)
	for {
		var stock pkg.SAPB1Response

		req, err := http.NewRequest(method, url+pagination, nil)
		if err != nil {
			fmt.Println("Request creation error:", err)
			return stocks, err
		}

		// req.Header.Add("Content-Type", "application/json")
		req.Header.Add("SessionId", pkg.SessionId)
		req.Header.Add("Cookie", fmt.Sprintf("B1SESSION=%s; ROUTEID=.node4", pkg.SessionId))

		res, err := pkg.Client.Do(req)
		if err != nil {
			fmt.Println("Request error:", err)
			return stocks, err
		}
		defer res.Body.Close()

		resByte, err := io.ReadAll(res.Body)
		if err != nil {
			return stocks, err
		}

		if err := json.Unmarshal(resByte, &stock); err != nil {
			fmt.Println("Unmarshal error:", err)
			return stocks, err
		}

		stocks = append(stocks, stock.Value...)

		if stock.OdataNextLink == "" {
			break
		}

		pagination = stock.OdataNextLink
		fmt.Println("PAGE", pagination)

		// break
		// fmt.Println("RESPONSE BODY: ", string(resByte))
	}

	return stocks, nil
}
