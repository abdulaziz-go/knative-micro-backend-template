package internal

import (
	"encoding/json"
	"fmt"
	"function/pkg"
	"io"
	"net/http"
)

func (h *Handler)CreateOrUpdateSubdivision() error {
	subdivisions, err := GetSubdivision()
	if err != nil {
		return fmt.Errorf("failed to get subdivisions: %w", err)
	}

	var (
		requestBody = pkg.UpsertManyReqBody{
			Data: pkg.UpsertManyData{
				Objects:   make([]map[string]interface{}, 0, len(subdivisions)),
				FieldSlug: "name",
			},
		}
	)

	for _, subdivision := range subdivisions {
		requestBody.Data.Objects = append(requestBody.Data.Objects, map[string]interface{}{
			"accountant":     pkg.GetStringValue(subdivision, "accountant"),
			"division_owner": pkg.GetStringValue(subdivision, "division_owner"),
			"name":           pkg.GetStringValue(subdivision, "name"),
		})

	}

	var url = "https://api.admin.u-code.io/v2/items/subdivision/upsert-many"
	if _, err := pkg.DoRequest(url, "POST", requestBody); err != nil {
		return fmt.Errorf("failed to create or update subdivisions: %w", err)
	}

	fmt.Println("Subdivision upsert completed successfully")
	return nil
}

func GetSubdivision() ([]map[string]interface{}, error) {
	var (
		pagination = "SQLQueries('OINVDEPGet')/List"
		url        = "https://212.83.166.117:50000/b1s/v1/"
		method     = "GET"

		subdivisions []map[string]interface{}
	)

	for {
		var subdivision pkg.SAPB1Response

		req, err := http.NewRequest(method, url+pagination, nil)
		if err != nil {
			fmt.Println("Request creation error:", err)
			return subdivisions, err
		}

		// req.Header.Add("Content-Type", "application/json")
		req.Header.Add("SessionId", pkg.SessionId)
		req.Header.Add("Cookie", fmt.Sprintf("B1SESSION=%s; ROUTEID=.node4", pkg.SessionId))

		res, err := pkg.Client.Do(req)
		if err != nil {
			fmt.Println("Request error:", err)
			return subdivisions, err
		}
		defer res.Body.Close()

		resByte, err := io.ReadAll(res.Body)
		if err != nil {
			return subdivisions, err
		}

		if err := json.Unmarshal(resByte, &subdivision); err != nil {
			fmt.Println("Unmarshal error:", err)
			return subdivisions, err
		}

		subdivisions = append(subdivisions, subdivision.Value...)

		if subdivision.OdataNextLink == "" {
			break
		}

		pagination = subdivision.OdataNextLink

		fmt.Println("RESPONSE BODY: ", string(resByte))
	}

	return subdivisions, nil
}
