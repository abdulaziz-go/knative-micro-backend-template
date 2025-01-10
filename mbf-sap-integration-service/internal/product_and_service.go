package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"function/pkg"
	"image/png"

	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
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
		collection    = h.MongoDB.Collection("product_and_services")
		operations    = []mongo.WriteModel{}
		erroredRows   = []map[string]interface{}{}
		erroredIssues = []string{}
	)

	for index, productAndService := range productAndServices {
		code := pkg.GetStringValue(productAndService, "ItemCode")
		existingDoc := collection.FindOne(context.Background(), bson.M{"code": code, "barcode": bson.M{"$ne": ""}})
		if existingDoc.Err() == nil {
			continue
		}

		qrPath := fmt.Sprintf("qrcodes/%s.png", code)
		err := qrGenerate(code, qrPath)
		if err != nil {
			erroredRows = append(erroredRows, productAndService)
			erroredIssues = append(erroredIssues, fmt.Sprintf("failed to generate QR code for code %s: %v", code, err))
			continue
		}

		qrURL, err := fileUpload(qrPath)
		if err != nil {
			erroredRows = append(erroredRows, productAndService)
			erroredIssues = append(erroredIssues, fmt.Sprintf("failed to upload QR code for code %s: %v", code, err))
			continue
		}

		itemGroupGuid := itemGroupId[pkg.GetIntValue(productAndService, "ItemsGroupCode")]
		U_direction := pkg.GetStringValue(productAndService, "U_direction")
		directionGuid := pkg.Directions[U_direction]
		name := productAndService["ItemName"]

		filter := bson.M{"code": code}
		update := bson.M{
			"$set": bson.M{
				"direction_id":   directionGuid,
				"direction_name": U_direction,
				"item_group_id":  itemGroupGuid,
				"code":           code,
				"name":           name,
				"barcode":        fmt.Sprintf("https://cdn.u-code.io/%v", qrURL),
			},
			"$setOnInsert": bson.M{
				"createdAt": time.Now(),
				"guid":      uuid.New().String(),
			},
		}
		operation := mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(update).
			SetUpsert(true)

		operations = append(operations, operation)
		fmt.Println("Index: ", index)
	}

	// Execute bulk write operation to update the documents in the collection.
	_, err = collection.BulkWrite(context.Background(), operations)
	if err != nil {
		return fmt.Errorf("error during bulk write: %w", err)
	}

	// Log and handle any errors that occurred during processing.
	if len(erroredRows) > 0 {
		h.Log.Warn().
			Interface("erroredRows", erroredRows).
			Strs("issues", erroredIssues).
			Msg("Errors occurred during product and service processing")
		return fmt.Errorf("errors occurred during processing: %v", erroredIssues)
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
func fileUpload(filePath string) (string, error) {
	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist at path: %s", filePath)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	multipartWriter := multipart.NewWriter(&requestBody)

	// Create a form file for the upload
	filePart, err := multipartWriter.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	// Copy the file content to the multipart writer
	if _, err := io.Copy(filePart, file); err != nil {
		return "", fmt.Errorf("failed to copy file content: %w", err)
	}

	// Close the multipart writer to finalize the form data
	if err := multipartWriter.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest(http.MethodPost, pkg.FileUploadURL, &requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	req.Header.Add("authorization", "API-KEY")
	req.Header.Add("X-API-KEY", pkg.AppId)
	// Send the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send HTTP request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == 201 {
		respByte, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		url := extractFilePath(respByte)
		return url, nil
	} else {
		return "", fmt.Errorf("failed to upload file: status code %d, response: %s", resp.StatusCode, resp.Status)
	}
}

func extractFilePath(respByte []byte) string {
	var result map[string]interface{}
	err := json.Unmarshal(respByte, &result)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return ""
	}
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		fmt.Println("Error: 'data' field not found or invalid")
		return ""
	}

	link, ok := data["link"].(string)
	if !ok {
		fmt.Println("Error: 'link' field not found or invalid")
		return ""
	}

	return link
}

func qrGenerate(code string, filePath string) error {
	qr, err := qrcode.New(code, qrcode.Medium)
	if err != nil {
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	err = png.Encode(file, qr.Image(256))
	if err != nil {
		return err
	}

	return nil
}
