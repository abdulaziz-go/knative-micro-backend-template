package internal

import (
	// "fmt"
	// "handler/function/config"
	// "handler/function/helper"
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
	// "github.com/google/uuid"
)

type documentLine struct {
	UDirection    string `json:"U_direction"`
	UDep          string `json:"U_dep"`
	ItemCode      string `json:"ItemCode"`
	Quantity      string `json:"Quantity"`
	WarehouseCode string `json:"WarehouseCode"`
	UnitPrice     string `json:"UnitPrice"`
}

type document struct {
	CardCode      string         `json:"CardCode"`
	DocDueDate    string         `json:"DocDueDate"`
	DocDate       string         `json:"DocDate"`
	UDirection    string         `json:"U_direction"`
	UDep          string         `json:"U_dep"`
	DocCurrency   string         `json:"DocCurrency"`
	DocumentLines []documentLine `json:"DocumentLines"`
}

func (h *Handler) CreateOrder(order pkg.Order) error {

	var (
		stocksIDs = map[string]int{}
		orderGuid = uuid.New().String()
		url       = pkg.SingleURL + "sale"
		reqBody   = pkg.Request{
			Data: map[string]interface{}{
				// "code":                      "", // must be generated
				// "created_date":              order.Data.CreatedDate,
				"delivery_address":          order.Data.DeliveryAddress,
				"created_date":              time.Now().Format(time.RFC3339),
				"currency_id":               pkg.Currency[order.Data.Currency],
				"direction_id":              order.Data.DirectionId,
				"direction_name":            order.Data.UDirection,
				"client_id":                 order.Data.ClientID,
				"total_sum":                 order.Data.TotalSumBeforeDiscount - order.Data.Discount,
				"discount":                  order.Data.Discount,
				"total_sum_before_discount": order.Data.TotalSumBeforeDiscount,
				"status":                    []string{"new"},
				"employee_id":               order.Data.EmployeeID,
				"warehouse_id":              order.Data.WarehouseId,
				"guid":                      orderGuid,
			},
		}

		orderItems = pkg.MultipleUpdateRequest{}
		document   = document{
			CardCode:      order.Data.CardCode,
			DocDueDate:    order.Data.DocDueDate,
			DocDate:       order.Data.DocDueDate,
			UDirection:    order.Data.UDirection,
			UDep:          order.Data.UDep,
			DocCurrency:   order.Data.Currency,
			DocumentLines: make([]documentLine, len(order.Data.OrderItems)),
		}
	)

	for index, item := range order.Data.OrderItems {
		stocksIDs[item.StockGuid] = item.Quantity

		document.DocumentLines[index] = documentLine{
			ItemCode:      item.ItemCode,
			Quantity:      fmt.Sprintf("%d", item.Quantity),
			WarehouseCode: item.WarehouseCode,
			UnitPrice:     fmt.Sprintf("%f", item.UnitPrice),
			UDirection:    item.DirectionName,
			UDep:          order.Data.UDep,
		}

		orderItems.Data.Objects = append(orderItems.Data.Objects, map[string]interface{}{
			"sale_id":                orderGuid,
			"direction_id":           item.DirectionId,
			"direction_name":         item.DirectionName,
			"currency_id":            pkg.Currency[order.Data.Currency],
			"sale_price":             item.UnitPrice,
			"warehouse_id":           item.WarehouseId,
			"quantity":               item.Quantity,
			"price":                  item.UnitPrice,
			"sold_quantity":          item.Quantity,
			"product_and_service_id": item.ProductAndServiceID,
			"stock_id":               item.StockGuid,
		})
	}

	requestBody, err := json.Marshal(document)
	if err != nil {
		return err
	}

	// SAP
	req, err := http.NewRequest("POST", "https://212.83.166.117:50000/b1s/v1/Orders", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	req.Header.Add("SessionId", pkg.SessionId)
	req.Header.Add("Cookie", fmt.Sprintf("B1SESSION=%s; ROUTEID=.node4", pkg.SessionId))

	resp, err := pkg.Client.Do(req)
	if err != nil {
		fmt.Println("Request error:", err)
		return err
	}

	respByte, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error on reading response body:", err)
	}

	var sapResponse = map[string]interface{}{}
	err = json.Unmarshal(respByte, &sapResponse)
	if err != nil {
		fmt.Println("Error on unmarshalling response body:", err)
		return err
	}

	// Update stock in ucode
	if err := h.updateStock(stocksIDs); err != nil {
		fmt.Println("error while updating stock", err)
		return err
	}

	// Ucode
	// fmt.Println("SAP RESPONSE: ", sapResponse)
	if sapResponse["error"] != nil {
		fmt.Println("SAP ORDER ERROR: ", sapResponse["error"])
		return fmt.Errorf("error while creating order: %v", sapResponse["error"])
	}
	fmt.Println("DOC NUM: ", sapResponse["DocNum"])
	reqBody.Data["code"] = sapResponse["DocNum"]
	_, err = pkg.DoRequest(url, "POST", reqBody)
	if err != nil {
		fmt.Println("Error on creating order in ucode:", err)
		return err
	}

	// fmt.Println("Ucode response: ", string(response))
	_, err = pkg.DoRequest(pkg.MultipleUpdateUrl+"sale_item", "PUT", orderItems)
	if err != nil {
		return fmt.Errorf("error on creating order items in ucode: %v", err)
	}
	fmt.Println("Order created successfully")
	return nil
}

func (h *Handler) updateStock(stocksIDs map[string]int) error {
	collection := h.MongoDB.Collection("stocks")
	// ctx := context.Background()

	for guid, quantity := range stocksIDs {
		if !isStockEnough(*collection, guid, quantity) {
			// if stock is not enough we need to make this order 2 step
			//
			return fmt.Errorf("stock with GUID %s is not enough", guid)

		}
	}

	return nil
}

func isStockEnough(collection mongo.Collection, guid string, quantity int) bool {
	// collection := db.Collection("stocks")
	ctx := context.Background()

	filter := bson.M{"guid": guid}

	var stock map[string]interface{}
	if err := collection.FindOne(ctx, filter).Decode(&stock); err != nil {
		fmt.Printf("Error while decoding stock with GUID %s: %v\n", guid, err)
		return false
	}

	stockQuantity, ok := stock["quantity"].(int)

	if !ok {
		fmt.Printf("Invalid quantity type for stock with GUID %s\n", guid)
		return false
	}

	if stockQuantity >= quantity {
		update := bson.M{"$set": bson.M{"quantity": stockQuantity - quantity}}
		_, err := collection.UpdateOne(ctx, filter, update)
		if err != nil {
			fmt.Printf("Error while updating stock with GUID %s: %v\n", guid, err)
			return false
		}
		return true
	}

	return false
}
