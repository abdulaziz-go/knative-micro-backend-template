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
	"go.mongodb.org/mongo-driver/mongo"
)

func (h *Handler) CreateOrUpdateWarehouse() error {
	if err := pkg.LoginSAP(); err != nil {
		return err
	}

	warehouses, err := getWarehouse()
	if err != nil {
		return fmt.Errorf("failed to get warehouses: %w", err)
	}

	collection := h.MongoDB.Collection("warehouses")
	for _, wh := range warehouses {
		fmt.Println(wh["WarehouseCode"])

		var subdivision map[string]interface{}
		{
			// find subdivision guid
			collection := h.MongoDB.Collection("subdivisions")
			filter := bson.M{
				"name": wh["U_dep"],
			}

			err := collection.FindOne(context.TODO(), filter).Decode(&subdivision)
			if err != nil && err != mongo.ErrNoDocuments {
				h.Log.Err(err).Msg("Error on finding subdivision")
				return err
			}

		}

		var (
			filter = bson.M{
				"code": wh["WarehouseCode"],
			}

			update = bson.M{
				"$set": bson.M{
					"updatedAt": time.Now(),
					"code":      wh["WarehouseCode"],
					"name":      wh["WarehouseName"],
				},
			}
		)
		result, err := collection.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			h.Log.Err(err).Msg("Error on updating warehouse")
			return err
		}

		if result.MatchedCount == 0 {
			_, err := collection.InsertOne(context.TODO(), bson.M{
				"guid":             uuid.New().String(),
				"subdivision_id":   subdivision["guid"],
				"subdivision_name": wh["U_dep"],
				"code":             wh["WarehouseCode"],
				"name":             wh["WarehouseName"],
				"createdAt":        time.Now(),
				"updatedAt":        time.Now(),
			})

			if err != nil {
				h.Log.Err(err).Msg("Error on inserting warehouse")
				return err
			}
		}
	}
	h.Log.Info().Msg("Warehouses created/updated successfully")
	return nil
}

func getWarehouse() ([]map[string]interface{}, error) {
	var (
		pagination = "Warehouses?$select=WarehouseName,WarehouseCode,U_dep"
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
