package internal

import (
	"encoding/json"
	"fmt"
	"function/pkg"

	"io"
	"net/http"
)

func (h *Handler)CreateModelCatIntoUcode(modelCats []map[string]interface{}) error {

	for _, modelCat := range modelCats {
		var (
			createModelCat    = pkg.SingleURL + "category_model"
			createModelCatReq = pkg.Request{
				Data: map[string]interface{}{
					"name": modelCat["Code"],
				},
			}
		)

		_, err := pkg.DoRequest(createModelCat, "POST", createModelCatReq)
		if err != nil {
			fmt.Println("Error on creating modelCat:", err)
			return err
		}

	}

	return nil
}

func GetModelCat() ([]map[string]interface{}, error) {
	var (
		pagination = "SQLQueries('MODELCATget')/List"
		url        = "https://212.83.166.117:50000/b1s/v1/"
		method     = "GET"
		modelCats  []map[string]interface{}
	)

	for {
		var modelCat pkg.SAPB1Response

		req, err := http.NewRequest(method, url+pagination, nil)
		if err != nil {
			fmt.Println("Request creation error:", err)
			return modelCats, err
		}

		// req.Header.Add("Content-Type", "application/json")
		req.Header.Add("SessionId", pkg.SessionId)
		req.Header.Add("Cookie", fmt.Sprintf("B1SESSION=%s; ROUTEID=.node4", pkg.SessionId))

		res, err := pkg.Client.Do(req)
		if err != nil {
			fmt.Println("Request error:", err)
			return modelCats, err
		}
		defer res.Body.Close()

		resByte, err := io.ReadAll(res.Body)
		if err != nil {
			return modelCats, err
		}

		if err := json.Unmarshal(resByte, &modelCat); err != nil {
			fmt.Println("Unmarshal error:", err)
			return modelCats, err
		}

		modelCats = append(modelCats, modelCat.Value...)

		if modelCat.OdataNextLink == "" {
			break
		}

		pagination = modelCat.OdataNextLink

		fmt.Println("RESPONSE BODY: ", string(resByte))
	}

	return modelCats, nil
}
