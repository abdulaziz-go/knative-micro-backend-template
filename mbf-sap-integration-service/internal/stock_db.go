package internal

import (
	"context"
	"database/sql"
	"fmt"
	"function/pkg"
	"log"
	"math/big"
	"time"

	_ "github.com/SAP/go-hdb/driver"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	dsn = "hdb://SYSTEM:2P1jYK3lIo@212.83.166.117:30015"
)

type productAndService struct {
	guid        string
	productName string
	directionId string
}

var (
	warehouses          = map[string][2]string{} // 0 warehouse guid, 1 subdivision guid
	productsAndServices = map[string]productAndService{}
)

func (h *Handler) UpdateStock() error {
	if err := pkg.LoginSAP(); err != nil {
		log.Fatal("error while login SAP", err)
		return err
	}

	if err := h.getProductAndWhs(); err != nil {
		fmt.Println("error while getting product and whs", err)
		return err
	}

	stocks, err := h.getDataFromSap()
	if err != nil {
		fmt.Println("error while getting data", err)
		return err
	}

	fmt.Println("LEN OF STOCKS: ", len(stocks))

	collectionStock := h.MongoDB.Collection("stocks")

	// fmt.Println("warehouses: ", len(stocks))
	//! 4.update or create stock
	for _, stock := range stocks {
		// fmt.Println(stock.ItemCode, stock.WhsCode, warehouses[stock.WhsCode])

		var (
			whsGuid, whsSubdivisionGuid = warehouses[stock.WhsCode][0], warehouses[stock.WhsCode][1]
			productData                 = productsAndServices[stock.ItemCode]
			filter                      = bson.M{
				"product_and_service_id": productData.guid,
				"warehouse_id":           whsGuid,
			}

			updateBody = bson.M{
				"$set": bson.M{
					"updatedAt":      time.Now(),
					"quantity":       stock.Quantity,
					"price":          stock.CostPrice,
					"product_name":   productData.productName,
					"direction_id":   productData.directionId,
					"subdivision_id": whsSubdivisionGuid,
				},
			}
		)

		result, err := collectionStock.UpdateOne(context.Background(), filter, updateBody)
		if err != nil {
			fmt.Println(err, " errors")
			return nil
		}
		// fmt.Println("matched count ", result.MatchedCount)

		if result.MatchedCount == 0 {
			// if data no available then create it
			createBody := bson.M{
				"guid":                   uuid.New().String(),
				"price":                  stock.CostPrice,
				"quantity":               stock.Quantity,
				"createdAt":              time.Now(),
				"updatedAt":              time.Now(),
				"product_code":           stock.ItemCode,
				"warehouse_id":           whsGuid,
				"warehouse_code":         stock.WhsCode,
				"product_and_service_id": productData.guid,
				"product_name":           productData.productName,
				"direction_id":           productData.directionId,
				"subdivision_id":         whsSubdivisionGuid,
			}

			collectionStock.InsertOne(context.Background(), createBody)
		}
	}
	return nil
}

func (h *Handler) getDataFromSap() ([]pkg.Stock, error) {
	var (
		dateParam = time.Now().Format("2006-01-02")
		offset    = 0
		stocks    []pkg.Stock
	)

	db, err := sql.Open("hdb", dsn)
	if err != nil {
		h.Log.Err(err).Msg("Failed to open the database")
		return nil, err

	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		h.Log.Err(err).Msg("Error connecting to the database")
		return nil, err
	}

	fmt.Println("Successfully connected to the SAP database")
	queryCount := `WITH STOCK_QTY1 AS (
					    SELECT 
					        T0."ItemCode",
					        T0."Warehouse",
					        SUM(T0."InQty" - T0."OutQty") AS "Qty"
					    FROM MBF_TEST1."OINM" T0
					    WHERE T0."DocDate" <= '2025-01-08'
					    GROUP BY 
					        T0."ItemCode", T0."Warehouse"
					),
					STOCK_PRICE1 AS (
					    SELECT  
					        T0."ItemCode", 
					        T0."LocCode" AS "WhsCode", 
					        SUM(T0."InQty" - T0."OutQty") AS "Qty", 
					        SUM(T0."SumStock") AS "Value",
					        CASE 
					            WHEN SUM(T0."InQty" - T0."OutQty") <> 0 THEN 
					                SUM(T0."SumStock") / SUM(T0."InQty" - T0."OutQty")  
					            ELSE 0 
					        END AS "CostPrice"
					    FROM MBF_TEST1."OIVL" T0
					    INNER JOIN MBF_TEST1."OITM" T1 ON T0."ItemCode" = T1."ItemCode"
					    WHERE T0."DocDate" <= '2025-01-08' 
					    GROUP BY T0."ItemCode", T0."LocCode"
					)
					SELECT COUNT(*)
					FROM (
					    SELECT DISTINCT 
					        A2."U_dep", 
					        A2."WhsCode", 
					        A0."ItemCode", 
					        A3."ItemName", 
					        A0."Qty" AS "Кол-во", 
					        A1."CostPrice" AS "Цена", 
					        A0."Qty" * A1."CostPrice" AS "Сумма" 
					    FROM STOCK_QTY1 A0 
					    INNER JOIN STOCK_PRICE1 A1 ON A1."ItemCode" = A0."ItemCode" AND A1."WhsCode" = A0."Warehouse"
					    INNER JOIN MBF_TEST1."OWHS" A2 ON A2."WhsCode" = A0."Warehouse"
					    INNER JOIN MBF_TEST1."OITM" A3 ON A3."ItemCode" = A0."ItemCode"
					    INNER JOIN MBF_TEST1."OGAR" A4 ON A4."ItmsGrpCod" = A3."ItmsGrpCod"
					    INNER JOIN MBF_TEST1."OITB" A5 ON A5."ItmsGrpCod" = A3."ItmsGrpCod"  
					    WHERE A0."Qty" <> 0 
					) AS SubQuery;`

	var count int
	err = db.QueryRow(queryCount).Scan(&count)
	if err != nil {
		log.Fatal("Failed to execute query:", err)
	}

	fmt.Println("Row Count:", count)
	for offset < count {
		sqlQuery := fmt.Sprintf(`
			WITH STOCK_QTY1 AS (
				SELECT 
					T0."ItemCode",
					T0."Warehouse",
					SUM(T0."InQty" - T0."OutQty") AS "Qty"
				FROM MBF_TEST1."OINM" T0
				WHERE T0."DocDate" <= '%s'
				GROUP BY 
					T0."ItemCode", T0."Warehouse"
			),
			STOCK_PRICE1 AS (
				SELECT  
					T0."ItemCode", T0."LocCode" AS "WhsCode", 
					SUM(T0."InQty" - T0."OutQty") AS "Qty", 
					SUM(T0."SumStock") AS "Value",
					CASE 
						WHEN SUM(T0."InQty" - T0."OutQty") <> 0 THEN 
							SUM(T0."SumStock") / SUM(T0."InQty" - T0."OutQty")  
						ELSE 0 
					END AS "CostPrice"
				FROM MBF_TEST1."OIVL" T0
				INNER JOIN MBF_TEST1."OITM" T1 ON T0."ItemCode" = T1."ItemCode"
				WHERE T0."DocDate" <= '%s' 
				GROUP BY T0."ItemCode", T0."LocCode"
			)
			SELECT DISTINCT 
				A2."U_dep", 
				A2."WhsCode", 
				A0."ItemCode", 
				A3."ItemName", 
				A0."Qty" AS "Кол-во", 
				A1."CostPrice" AS "Цена", 
				A0."Qty" * A1."CostPrice" AS "Сумма" 
			FROM STOCK_QTY1 A0 
			INNER JOIN STOCK_PRICE1 A1 ON A1."ItemCode" = A0."ItemCode" AND A1."WhsCode" = A0."Warehouse"
			INNER JOIN MBF_TEST1."OWHS" A2 ON A2."WhsCode" = A0."Warehouse"
			INNER JOIN MBF_TEST1."OITM" A3 ON A3."ItemCode" = A0."ItemCode"
			INNER JOIN MBF_TEST1."OGAR" A4 ON A4."ItmsGrpCod" = A3."ItmsGrpCod"
			INNER JOIN MBF_TEST1."OITB" A5 ON A5."ItmsGrpCod" = A3."ItmsGrpCod"  
			WHERE A0."Qty" <> 0  LIMIT 10 OFFSET %d`, dateParam, dateParam, offset)
		offset += 10
		fmt.Println("OFFSET: ", offset)
		// fmt.Println(sqlQuery)
		// break
		rows, err := db.Query(sqlQuery)
		if err != nil {
			log.Fatal("Error executing the query: ", err)
			return nil, err

		}
		defer rows.Close()

		var (
			U_dep                       = sql.NullString{}
			WhsCode, ItemCode, ItemName string
			Qty, CostPrice, Summa       *big.Rat
		)

		// if !rows.NextResultSet() {
		// 	break
		// }

		for rows.Next() {
	
			err := rows.Scan(
				&U_dep,
				&WhsCode,
				&ItemCode,
				&ItemName,
				&Qty,
				&CostPrice,
				&Summa,
			)
			if err != nil {
				log.Fatal("Error scanning row: ", err)
				return nil, err
			}

			// fmt.Println(ItemCode)

			if ItemCode == "" && ItemName == "" {
				break
			}

			var (
				qty, _       = Qty.Float64()
				costPrice, _ = CostPrice.Float64()
				summa, _     = Summa.Float64()
				u_dep        = U_dep.String
			)

			// fmt.Println(
			// 	u_dep,
			// 	WhsCode,
			// 	ItemCode,
			// 	ItemName,
			// 	qty,
			// 	costPrice,
			// 	summa,
			// )

			stocks = append(stocks, pkg.Stock{
				Bulim:     u_dep,
				WhsCode:   WhsCode,
				ItemCode:  ItemCode,
				ItemName:  ItemName,
				Quantity:  qty,
				CostPrice: costPrice,
				Summa:     summa,
			})

		}

		if err := rows.Err(); err != nil {
			log.Fatal("Error reading rows: ", err)
			return nil, err

		}

	}

	return stocks, nil
}

func (h *Handler) getProductAndWhs() error {
	whsColl := h.MongoDB.Collection("warehouses")
	productAS := h.MongoDB.Collection("product_and_services")

	//! 2. get all warehouses
	cursor, err := whsColl.Find(context.Background(), bson.M{})
	if err != nil {
		fmt.Println("error while getting warehouses", err)
		return nil
	}

	for cursor.Next(context.Background()) {
		var document bson.M
		if err := cursor.Decode(&document); err != nil {
			log.Printf("Error decoding document: %v", err)
			continue
		}

		var code = pkg.GetStringValue(document, "code")
		warehouses[code] = [2]string{
			0: pkg.GetStringValue(document, "guid"),
			1: pkg.GetStringValue(document, "subdivision_id"),
		}

	}

	//! 3. get all products_and_services
	cursor, err = productAS.Find(context.Background(), bson.M{})
	if err != nil {
		fmt.Println("error while getting warehouses", err)
		return nil
	}

	for cursor.Next(context.Background()) {
		var document bson.M
		if err := cursor.Decode(&document); err != nil {
			log.Printf("Error decoding document: %v", err)
			continue
		}
		// fmt.Println("DOCUMENT: ", document)
		var (
			code        = pkg.GetStringValue(document, "code")
			guid        = pkg.GetStringValue(document, "guid")
			productName = pkg.GetStringValue(document, "name")
			directionId = pkg.GetStringValue(document, "direction_id")
		)
		// fmt.Println("directionId", directionId)
		productsAndServices[code] = productAndService{
			guid:        guid,
			productName: productName,
			directionId: directionId,
		}

	}
	return nil
}
