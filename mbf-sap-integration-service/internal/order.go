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

// 1. check cases is it razovey klient if so then продажа +- оплата in sap table else just продажа
// 2. check if stock is enough for 2 step order
func (h *Handler) CreateOfflineOrder(order *pkg.Order) error {
	if err := pkg.LoginSAP(); err != nil {
		h.Log.Err(err).Msg("Error on login SAP")
		return err
	}

	if order.PaymentType == "cash" {
		// create order in sap
		if err := createOrderInSAP(order); err != nil {
			fmt.Println("Error while creating order in SAP:", err)
			return err
		}
	}

	// var (
	// 	orderGuid = uuid.New().String()
	// 	orderBody = map[string]interface{}{
	// 		"guid":                      orderGuid,
	// 		"status":                    []string{"new"},
	// 		"total_sum_before_discount": order.TotalSumBeforeDiscount,
	// 		"discount":                  order.Discount,
	// 		"client_id":                 order.ClientID,
	// 		"direction_id":              order.DirectionID,
	// 		"subdivision_id":            order.SubdivisionID,
	// 		"created_date":              time.Now().Format(time.RFC3339),
	// 		"currency_id":               pkg.Currency[order.Currency],
	// 		"delivery_address":          order.DeliveryAddress,
	// 		"employee_id":               order.EmployeeID,
	// 		"total_quantity":            order.TotalQuantity,
	// 		"payment_type":              order.PaymentType, // is cash and client is razovey klient

	// 		// "step":1/2, // if warehouse quantity less than order quantity then it is 2 step order
	// 		// "code":"",// sap code, after insertin in sap must be added
	// 		// "payment_amount":            order.PaymentAmount, // backend formula in ucode
	// 		// "total_sum":                 order.TotalSumBeforeDiscount - order.Discount, // this is formula frontend in ucode
	// 	}
	// )

	return nil

}



func createOrderInSAP(order *pkg.Order) error {
	document := document{
		CardCode:      order.CardCode,
		DocDueDate:    order.DocDueDate,
		DocDate:       order.DocDueDate,
		UDirection:    order.DirectionName,
		UDep:          order.UDep,
		DocCurrency:   order.Currency,
		DocumentLines: make([]documentLine, len(order.OrderItems)),
	}

	for index, item := range order.OrderItems {
		document.DocumentLines[index] = documentLine{
			ItemCode:      item.ItemCode,
			WarehouseCode: item.WarehouseCode,
			UnitPrice:     fmt.Sprintf("%f", item.UnitPrice),
			UDep:          order.SubdivisionName,
		}
	}

	requestBody, err := json.Marshal(document)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://212.83.166.117:50000/b1s/v1/Orders", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	req.Header.Add("SessionId", pkg.SessionId)
	req.Header.Add("Cookie", fmt.Sprintf("B1SESSION=%s; ROUTEID=.node4", pkg.SessionId))

	resp, err := pkg.Client.Do(req)
	if err != nil {
		return err
	}

	respByte, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var sapResponse = map[string]interface{}{}
	err = json.Unmarshal(respByte, &sapResponse)
	if err != nil {
		return err
	}

	if sapResponse["error"] != nil {
		return fmt.Errorf("error while creating order: %v", sapResponse["error"])
	}

	return nil
}

func (h *Handler) isStockEnough(stocksIDs map[string]int) bool {
	collection := h.MongoDB.Collection("stocks")
	isEnough := true
	for guid, quantity := range stocksIDs {
		var (
			ctx    = context.Background()
			filter = bson.M{"guid": guid}
			stock  map[string]interface{}
		)

		if err := collection.FindOne(ctx, filter).Decode(&stock); err != nil {
			h.Log.Err(err).Msgf("Error while getting stock with GUID %s", guid)
			return false
		}

		stockQuantity, ok := stock["quantity"].(int)
		if !ok {
			fmt.Printf("Invalid quantity type for stock with GUID %s\n", guid)
			return false
		}

		if stockQuantity >= quantity { // current stock amount is enough for order, just update stock
			update := bson.M{"$set": bson.M{"quantity": stockQuantity - quantity}}
			_, err := collection.UpdateOne(ctx, filter, update)
			if err != nil {
				h.Log.Err(err).Msgf("Error while updating stock with GUID %s", guid)
				return false
			}

		} else {
			isEnough = false
		}
	}

	return isEnough
}

func (h *Handler) CreateOrder(order *pkg.Order) error {

	var (
		// stocksIDs = map[string]int{}
		orderGuid = uuid.New().String()
		url       = pkg.SingleURL + "sale"
		reqBody   = pkg.Request{
			Data: map[string]interface{}{
				"delivery_address":          order.DeliveryAddress,
				"created_date":              time.Now().Format(time.RFC3339),
				"currency_id":               pkg.Currency[order.Currency],
				"direction_id":              order.DirectionID,
				"direction_name":            order.DirectionName,
				"client_id":                 order.ClientID,
				"total_sum":                 order.TotalSumBeforeDiscount - order.Discount,
				"discount":                  order.Discount,
				"total_sum_before_discount": order.TotalSumBeforeDiscount,
				"status":                    []string{"new"},
				"employee_id":               order.EmployeeID,
				"guid":                      orderGuid,
				// "code":                      "", // must be generated
				// "created_date":              order.Data.CreatedDate,
				// "warehouse_id":              order.Data.WarehouseId,
			},
		}

		orderItems = pkg.MultipleUpdateRequest{}
		document   = document{
			CardCode:      order.CardCode,
			DocDueDate:    order.DocDueDate,
			DocDate:       order.DocDueDate,
			UDirection:    order.DirectionName,
			UDep:          order.UDep,
			DocCurrency:   order.Currency,
			DocumentLines: make([]documentLine, len(order.OrderItems)),
		}
	)

	for index, item := range order.OrderItems {
		// stocksIDs[item.StockGuid] = item.Quantity

		document.DocumentLines[index] = documentLine{
			ItemCode:      item.ItemCode,
			WarehouseCode: item.WarehouseCode,
			UnitPrice:     fmt.Sprintf("%f", item.UnitPrice),
			UDep:          order.UDep,
			// UDirection:    item.DirectionName,
			// Quantity:      fmt.Sprintf("%d", item.Quantity),
		}

		orderItems.Data.Objects = append(orderItems.Data.Objects, map[string]interface{}{
			"price":                  item.UnitPrice,
			"sale_id":                orderGuid,
			"sale_price":             item.UnitPrice,
			"currency_id":            pkg.Currency[order.Currency],
			"product_and_service_id": item.ProductAndServiceID,
			// "direction_id":           item.DirectionId,
			// "direction_name":         item.DirectionName,
			// "warehouse_id":           item.WarehouseId,
			// "quantity":               item.Quantity,
			// "sold_quantity":          item.Quantity,
			// "stock_id":               item.StockGuid,
		})
	}

	// if err := h.updateStock(stocksIDs); err != nil {
	// 	fmt.Println("error while updating stock", err)
	// 	return err
	// }

	_, err := pkg.DoRequest(url, "POST", reqBody)
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