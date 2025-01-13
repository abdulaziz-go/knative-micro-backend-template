// this is ItemGroups in sap
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
)

func (h *Handler) ProductsCronJob() error {
	if err := pkg.LoginSAP(); err != nil {
		h.Log.Err(err).Msg("Error on login SAP ProductsCronJob")
		return err
	}

	products, err := getProduct()
	if err != nil {
		return fmt.Errorf("failed to get products: %w", err)

	}

	var (
		collection = h.MongoDB.Collection("products")
		createBody = []interface{}{}
	)

	for _, itemGroup := range products {
		var (
			name   = pkg.GetStringValue(itemGroup, "Code")
			filter = bson.M{
				"name": name,
			}

			updateBody = bson.M{
				"$set": bson.M{
					"name":      name,
					"updatedAt": time.Now(),
				},
			}
		)

		result, err := collection.UpdateOne(context.Background(), filter, updateBody)
		if err != nil {
			fmt.Println(err, " errors")
			return err
		}

		if result.MatchedCount == 0 {
			createBody = append(createBody, bson.M{
				"name":      name,
				"updatedAt": time.Now(),
				"createdAt": time.Now(),
				"guid":      uuid.New().String(),
			})
		}
	}

	_, err = collection.InsertMany(context.Background(), createBody)
	if err != nil {
		return fmt.Errorf("failed to insert products: %w", err)
	}
	fmt.Println("CreateProduct CRONJOB SUCCESSFULLY WORKED")
	return nil
}

func getProduct() ([]map[string]interface{}, error) {
	var (
		pagination = "SQLQueries('Get@Items')/List"
		url        = "https://212.83.166.117:50000/b1s/v1/"
		method     = "GET"
		products   []map[string]interface{}
	)

	for {
		var product pkg.SAPB1Response

		req, err := http.NewRequest(method, url+pagination, nil)
		if err != nil {
			fmt.Println("Request creation error:", err)
			return products, err
		}

		// req.Header.Add("Content-Type", "application/json")
		req.Header.Add("SessionId", pkg.SessionId)
		req.Header.Add("Cookie", fmt.Sprintf("B1SESSION=%s; ROUTEID=.node4", pkg.SessionId))

		res, err := pkg.Client.Do(req)
		if err != nil {
			fmt.Println("Request error:", err)
			return products, err
		}
		defer res.Body.Close()

		resByte, err := io.ReadAll(res.Body)
		if err != nil {
			return products, err
		}

		if err := json.Unmarshal(resByte, &product); err != nil {
			fmt.Println("Unmarshal error:", err)
			return products, err
		}

		products = append(products, product.Value...)

		if product.OdataNextLink == "" {
			break
		}

		pagination = product.OdataNextLink
		fmt.Println("OdataNextLink: ", product.OdataNextLink)
		// fmt.Println("RESPONSE BODY: ", string(resByte))
	}

	return products, nil
}
