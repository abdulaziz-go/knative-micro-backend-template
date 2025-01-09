// InventoryTransferRequests api -> StockTransfers send docEntry it comes from previous response
package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"function/pkg"
	"io"
	"net/http"

	"github.com/google/uuid"
)

const (
	sapInventoryTransferURL = "https://212.83.166.117:50000/b1s/v2/InventoryTransferRequests"
	sapStockTransferURL     = "https://212.83.166.117:50000/b1s/v2/StockTransfers"
	movementEndpoint        = "movement"
	movementItemEndpoint    = "movement_item"
	contentTypeJSON         = "application/json"
	cookieFormat            = "B1SESSION=%s; ROUTEID=.node4"
)

// MovementRequest handles the inventory movement request process
func (h *Handler) MovementRequest(w http.ResponseWriter, r *http.Request) {
	// Step 1: Authenticate with SAP
	if err := pkg.LoginSAP(); err != nil {
		h.HandleError(w, err, "Failed to login to SAP")
		return
	}

	// Step 2: Decode the incoming request
	var movement pkg.MovementRequest
	if err := json.NewDecoder(r.Body).Decode(&movement); err != nil {
		h.HandleError(w, err, "Invalid request body for MovementRequest")
		return
	}

	switch movement.MovementType {
	case "SEND":

		if err := stockTransferSAP(movement); err != nil {
			h.HandleError(w, err, "Failed to send data to SAP in MovementRequest")
			return
		}

		if err := h.updateMovementStatus(movement.GUID, "delevered"); err != nil {

		}
		pkg.HandleResponse(w, map[string]interface{}{"data": "OK"}, http.StatusCreated)
		return

	case "IN_ONE":
		if err := stockTransferSAP(movement); err != nil {
			h.HandleError(w, err, "Failed to send data to SAP in MovementRequest")
			return
		}

		if err := sendToUcode(movement, 0, 0); err != nil {

		}
		pkg.HandleResponse(w, map[string]interface{}{"data": "OK"}, http.StatusCreated)
		return

	}

	// Step 3: Send transferInventory to SAP
	docEntry, docNum, err := transferInventorySAP(movement)
	if err != nil {
		h.HandleError(w, err, "Failed to send data to SAP in MovementRequest")
		return
	}
	// Step 4: Send data to Ucode
	if err := sendToUcode(movement, docEntry, docNum); err != nil {
		h.HandleError(w, err, "Failed to send data to Ucode in MovementRequest")
		return
	}

	pkg.HandleResponse(w, map[string]interface{}{"message": "OK", "code": http.StatusOK}, http.StatusOK)
}

func (h *Handler) updateMovementStatus(movementID string, status string) error {
	collection := h.MongoDB.Collection("movements")
	_, err := collection.UpdateOne(
		context.TODO(),
		map[string]interface{}{"guid": movementID},
		map[string]interface{}{"$set": map[string]interface{}{"status": []string{status}}},
	)
	if err != nil {
		return fmt.Errorf("failed to update movement status: %w", err)
	}

	return nil
}

func stockTransferSAP(movement pkg.MovementRequest) error {
	// Construct the request body
	requestBody, err := createSAPRequestBody(movement)
	if err != nil {
		return fmt.Errorf("failed to create SAP request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, sapStockTransferURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create SAP HTTP request: %w", err)
	}
	req.Header.Add("SessionId", pkg.SessionId)
	req.Header.Add("Content-Type", contentTypeJSON)
	req.Header.Add("Cookie", fmt.Sprintf(cookieFormat, pkg.SessionId))

	// Send the request
	res, err := pkg.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SAP request: %w", err)
	}
	defer res.Body.Close()

	return nil
}

// transferInventorySAP handles sending movement data to SAP
func transferInventorySAP(movement pkg.MovementRequest) (int, int, error) {
	// Construct the request body
	requestBody, err := createSAPRequestBody(movement)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create SAP request body: %w", err)
	}

	fmt.Println(string(requestBody))
	// Create the HTTP request
	req, err := http.NewRequest(http.MethodPost, sapInventoryTransferURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create SAP HTTP request: %w", err)
	}

	req.Header.Add("SessionId", pkg.SessionId)
	req.Header.Add("Content-Type", contentTypeJSON)
	req.Header.Add("Cookie", fmt.Sprintf(cookieFormat, pkg.SessionId))

	// Send the request
	res, err := pkg.Client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to send SAP request: %w", err)
	}
	defer res.Body.Close()

	// Log the response
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read SAP response: %w", err)
	}

	var data = map[string]interface{}{}
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, 0, fmt.Errorf("failed to unmarshal SAP response: %w", err)
	}

	docEntry, ok := data["DocEntry"].(float64)
	if !ok {
		return 0, 0, fmt.Errorf("failed to get DocEntry from SAP response")
	}

	docNum, ok := data["DocNum"].(float64)
	if !ok {
		return 0, 0, fmt.Errorf("failed to get DocNum from SAP response")
	}

	return int(docEntry), int(docNum), nil
}

// createSAPRequestBody generates the JSON body for the SAP request
func createSAPRequestBody(movement pkg.MovementRequest) ([]byte, error) {
	requestBody := map[string]interface{}{
		"DocDate":       movement.CreatedDate,
		"DueDate":       movement.CreatedDate,
		"U_direction":   movement.DirectionName,
		"U_dep":         movement.SubdivisionName,
		"FromWarehouse": movement.FromWarehouseCode,
		"ToWarehouse":   movement.ToWarehouseCode,
	}

	// "BaseType":"InventoryTransferRequest",
	// "BaseLine":0...
	// "BaseEntry"

	// Add stock transfer lines
	stockTransferLines := []map[string]interface{}{}
	for idx, item := range movement.Items {
		data := map[string]interface{}{
			"LineNum":           idx,
			"ItemCode":          item.ItemCode,
			"Quantity":          item.Quantity,
			"WarehouseCode":     item.ToWarehouseCode,
			"FromWarehouseCode": item.FromWarehouseCode,
		}

		if movement.MovementType == "SEND" {
			data["BaseType"] = "InventoryTransferRequest"
			data["BaseLine"] = idx
			data["BaseEntry"] = movement.BaseEntry
		}

		stockTransferLines = append(stockTransferLines, data)
	}
	requestBody["StockTransferLines"] = stockTransferLines

	return json.Marshal(requestBody)
}

// sendToUcode handles sending movement data to Ucode
func sendToUcode(movement pkg.MovementRequest, docEntry, docNum int) error {
	// Construct the movement request
	movementGuid := uuid.New().String()
	movementReqBody := pkg.Request{
		Data: map[string]interface{}{
			"guid":             movementGuid,
			"code":             docNum,
			"docEntry":         docEntry,
			"direction_id":     movement.DirectionID,
			"subdivision_id":   movement.SubdivisionID,
			"warehouse_id":     movement.FromWarehouseID,
			"warehouse_id_2":   movement.ToWarehouseID,
			"status":           []string{"new"},
			"client_id":        movement.ClientID,
			"subdivision_id_2": movement.SubdivisionID2,
			"employee_id":      movement.EmployeeID,
		},
	}

	// Send the movement request
	movementURL := pkg.SingleURL + movementEndpoint
	if _, err := pkg.DoRequest(movementURL, http.MethodPost, movementReqBody); err != nil {
		return fmt.Errorf("failed to send movement request to Ucode: %w", err)
	}

	// Construct and send movement items
	movementItems := pkg.MultipleUpdateRequest{}
	for _, item := range movement.Items {
		movementItems.Data.Objects = append(movementItems.Data.Objects, map[string]interface{}{
			"stock_id":       item.StockID,
			"quantity":       item.Quantity,
			"movement_id":    movementGuid,
			"warehouse_id":   item.FromWarehouseID,
			"warehouse_id2":  item.ToWarehouseID,
			"subdivision_id": movement.SubdivisionID,
		})
	}

	movementItemURL := pkg.MultipleUpdateUrl + movementItemEndpoint
	if _, err := pkg.DoRequest(movementItemURL, http.MethodPut, movementItems); err != nil {
		return fmt.Errorf("failed to send movement items to Ucode: %w", err)
	}

	return nil
}
