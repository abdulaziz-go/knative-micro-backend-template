// this is ItemGroups in sap
package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"function/pkg"

	"time"

	"io"
	"net/http"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

func (h *Handler) CreateItemGroup() error {
	if err := pkg.LoginSAP(); err != nil {
		h.Log.Err(err).Msg("Error on login SAP ItemGroupCronjob")
		return err
	}

	itemGroups, err := getItemGroup()
	if err != nil {
		return fmt.Errorf("failed to get itemGroups: %w", err)

	}
	fmt.Println(len(itemGroups))

	var collection = h.MongoDB.Collection("item_groups")

	for _, itemGroup := range itemGroups {
		var (
			filter = bson.M{
				"number": itemGroup["Number"],
			}

			updateBody = bson.M{
				"$set": bson.M{
					"updatedAt":  time.Now(),
					"group_name": itemGroup["GroupName"],
					"number":     itemGroup["Number"],
				},
			}
		)

		result, err := collection.UpdateOne(context.Background(), filter, updateBody)
		if err != nil {
			fmt.Println(err, " errors")
			return err
		}

		if result.MatchedCount == 0 {
			var createBody = bson.M{
				"guid":       uuid.New().String(),
				"updatedAt":  time.Now(),
				"createdAt":  time.Now(),
				"group_name": itemGroup["GroupName"],
				"number":     itemGroup["Number"],
			}

			collection.InsertOne(context.Background(), createBody)

		}
	}

	fmt.Println("ItemGroups CRONJOB SUCCESSFULLY WORKED")
	return nil
}

func getItemGroup() ([]map[string]interface{}, error) {
	var (
		pagination = "ItemGroups?$select=Number,GroupName"
		url        = "https://212.83.166.117:50000/b1s/v1/"
		method     = "GET"

		itemGroups []map[string]interface{}
	)

	for {
		var itemGroup pkg.SAPB1Response

		req, err := http.NewRequest(method, url+pagination, nil)
		if err != nil {
			fmt.Println("Request creation error:", err)
			return itemGroups, err
		}

		// req.Header.Add("Content-Type", "application/json")
		req.Header.Add("SessionId", pkg.SessionId)
		req.Header.Add("Cookie", fmt.Sprintf("B1SESSION=%s; ROUTEID=.node4", pkg.SessionId))

		res, err := pkg.Client.Do(req)
		if err != nil {
			fmt.Println("Request error:", err)
			return itemGroups, err
		}
		defer res.Body.Close()

		resByte, err := io.ReadAll(res.Body)
		if err != nil {
			return itemGroups, err
		}

		if err := json.Unmarshal(resByte, &itemGroup); err != nil {
			fmt.Println("Unmarshal error:", err)
			return itemGroups, err
		}

		itemGroups = append(itemGroups, itemGroup.Value...)

		if itemGroup.OdataNextLink == "" {
			break
		}
		fmt.Println("itemGroup.OdataNextLink: ", itemGroup.OdataNextLink)
		pagination = itemGroup.OdataNextLink

		// fmt.Println("RESPONSE BODY: ", string(resByte))
	}

	return itemGroups, nil
}
