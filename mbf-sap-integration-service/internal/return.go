package internal

import (
	"fmt"
	"function/pkg"
)

func (h *Handler) ReturnProduct(returnProducts pkg.ReturnData) error {

	var (
		url     = pkg.SingleURL + "sale"
		reqBody = pkg.Request{
			Data: map[string]interface{}{
				"guid": returnProducts.SaleID,
			},
		}
		orderItems = pkg.MultipleUpdateRequest{}
	)

	for _, item := range returnProducts.ReturnItems {
		orderItems.Data.Objects = append(orderItems.Data.Objects, map[string]interface{}{
			"guid":                 item.Guid,
			"return_reason":        returnProducts.ReturnReason,
			"client_took_quantity": "", // ?
			"returned_quantity":    item.ReturnQuantity,
		})
	}

	// fmt.Println("Ucode response: ", string(response))
	_, err := pkg.DoRequest(url, "PUT", reqBody)
	if err != nil {
		return fmt.Errorf("error on creating order items in ucode: %v", err)
	}

	// fmt.Println("Ucode response: ", string(response))
	_, err = pkg.DoRequest(pkg.MultipleUpdateUrl+"sale_item", "PUT", orderItems)
	if err != nil {
		return fmt.Errorf("error on creating order items in ucode: %v", err)
	}
	fmt.Println("Successfully returnProduct")
	return nil
}

func ReturnSAP() error {
	// TODO: implement this function
	return nil
}
