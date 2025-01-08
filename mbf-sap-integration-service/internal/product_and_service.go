package internal

import (
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

var itemGroupId = map[int]string{}

// code
func (h *Handler) ProductAndServiceCronJob() error {
	if err := pkg.LoginSAP(); err != nil {
		h.Log.Err(err).Msg("Error on login SAP ProductAndServiceCronJob")
		return err
	}

	if err := h.itemGroup(); err != nil {
		return fmt.Errorf("failed to get item group: %w", err)
	}

	productAndServices, err := getProductAndServices()
	if err != nil {
		return fmt.Errorf("failed to get product and services: %w", err)
	}

	var (
		collection = h.MongoDB.Collection("product_and_services")
		operations = []mongo.WriteModel{}
	)

	for index, productAndService := range productAndServices {
		var (
			itemGroupGuid = itemGroupId[pkg.GetIntValue(productAndService, "ItemsGroupCode")]
			U_direction   = pkg.GetStringValue(productAndService, "U_direction")
			directionGuid = pkg.Directions[U_direction]
			code          = pkg.GetStringValue(productAndService, "ItemCode")
			name          = productAndService["ItemName"]
			filter        = bson.M{"code": code}
			update        = bson.M{
				"$set": bson.M{
					"direction_id":   directionGuid,
					"direction_name": U_direction,
					"item_group_id":  itemGroupGuid,
					"code":           code,
					"name":           name,
				},
				"$setOnInsert": bson.M{
					"createdAt": time.Now(),
					"guid":      uuid.New().String(),
				},
			}
			operation = mongo.NewUpdateOneModel().
					SetFilter(filter).
					SetUpdate(update).
					SetUpsert(true)
		)

		operations = append(operations, operation)
		fmt.Println("Index: ", index)
	}

	// Execute bulk write
	_, err = collection.BulkWrite(context.Background(), operations)
	if err != nil {
		return fmt.Errorf("error during bulk write: %w", err)
	}

	fmt.Println("Successfully updated and inserted product and services.")

	return nil
}

func (h *Handler) itemGroup() error {

	collection := h.MongoDB.Collection("item_groups")
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		fmt.Println("error while getting item group: ", err)
		return err
	}

	for cursor.Next(context.Background()) {
		var mongoData = map[string]interface{}{}
		if err := cursor.Decode(&mongoData); err != nil {
			fmt.Println("error while decoding mongo data: ", err)
			return err
		}

		var (
			number = pkg.GetIntValue(mongoData, "number")
			guid   = pkg.GetStringValue(mongoData, "guid")
		)
		itemGroupId[number] = guid
	}

	return nil

}

func getProductAndServices() ([]map[string]interface{}, error) {
	var (
		pagination = "Items?$select=ItemCode,ItemName,ItemsGroupCode,U_direction"
		url        = "https://212.83.166.117:50000/b1s/v1/"
		method     = "GET"

		productAndServices []map[string]interface{}
	)

	for {
		var productAndService pkg.SAPB1Response

		req, err := http.NewRequest(method, url+pagination, nil)
		if err != nil {
			fmt.Println("Request creation error:", err)
			return productAndServices, err
		}

		// req.Header.Add("Content-Type", "application/json")
		req.Header.Add("SessionId", pkg.SessionId)
		req.Header.Add("Cookie", fmt.Sprintf("B1SESSION=%s; ROUTEID=.node4", pkg.SessionId))

		res, err := pkg.Client.Do(req)
		if err != nil {
			fmt.Println("Request error:", err)
			return productAndServices, err
		}
		defer res.Body.Close()

		resByte, err := io.ReadAll(res.Body)
		if err != nil {
			return productAndServices, err
		}

		if err := json.Unmarshal(resByte, &productAndService); err != nil {
			fmt.Println("Unmarshal error:", err)
			return productAndServices, err
		}

		productAndServices = append(productAndServices, productAndService.Value...)

		if productAndService.OdataNextLink == "" {
			break
		}

		pagination = productAndService.OdataNextLink

		fmt.Println("PAGINATION: ", pagination)
	}

	return productAndServices, nil
}
