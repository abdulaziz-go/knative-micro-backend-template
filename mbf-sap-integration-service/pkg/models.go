package pkg

import (
	"os"

	cache "github.com/golanguzb70/redis-cache"
	"github.com/rs/zerolog"
)

type Params struct {
	CacheClient    cache.RedisCache
	CacheAvailable bool
	Log            zerolog.Logger
	Config         *Config
}

func NewParams(cfg *Config) *Params {
	response := &Params{
		Config: cfg,
	}

	response.Log = zerolog.New(os.Stdout).With().Any("function", cfg.Name).Logger()

	if cfg.Redis.Enabled {
		cacheConfig := &cache.Config{
			RedisHost:     cfg.Redis.RedisHost,
			RedisPort:     cfg.Redis.RedisPort,
			RedisUsername: cfg.Redis.RedisUser,
			RedisPassword: cfg.Redis.RedisPass,
		}

		cacheClient, err := cache.New(cacheConfig)
		if err != nil {
			response.Log.Error().Msgf("Error creating cache client: %v", err)
			response.CacheAvailable = false
		} else {
			response.CacheClient = cacheClient
			response.CacheAvailable = true
		}
	}

	return response
}

type SAPB1Response struct {
	OdataMetadata string                   `json:"odata.metadata"`
	SqlText       string                   `json:"SqlText"`
	Value         []map[string]interface{} `json:"value"`
	OdataNextLink string                   `json:"odata.nextLink"`
}

type LoginSAPResponse struct {
	OdataMetadata  string `json:"odata.metadata"`
	SessionId      string `json:"SessionId"`
	Version        string `json:"Version"`
	SessionTimeout int    `json:"SessionTimeout"`
}

type Request struct {
	Data map[string]interface{} `json:"data"`
}

type GetListClientApiResponse struct {
	Data GetListClientApiData `json:"data"`
}

type GetListClientApiData struct {
	Data GetListClientApiResp `json:"data"`
}

type GetListClientApiResp struct {
	Response []map[string]interface{} `json:"response"`
}

type Stock struct {
	Bulim     string
	WhsCode   string
	ItemCode  string
	ItemName  string
	Quantity  float64
	CostPrice float64
	Summa     float64
}

type Order struct {
	OrderID                string       `json:"order_id"`
	ClientID               string       `json:"client_id"`
	EmployeeID             string       `json:"employee_id"`
	CardCode               string       `json:"CardCode"`
	DocDueDate             string       `json:"DocDueDate"`
	DocDate                string       `json:"DocDate"`
	UDep                   string       `json:"U_dep"`
	Currency               string       `json:"currency"`
	TotalSumBeforeDiscount float64      `json:"total_sum_before_discount"`
	DirectionName          string       `json:"direction_name"`
	Discount               float64      `json:"discount"`
	DeliveryAddress        string       `json:"delivery_address"`
	OrderItems             []OrderItems `json:"order_items"`
	DirectionID            string       `json:"direction_id"`
	SubdivisionName        string       `json:"subdivision_name"`
	SubdivisionID          string       `json:"subdivision_id"`
	TotalQuantity          int          `json:"total_quantity"`
	PaymentType            string       `json:"payment_type"`
}
type OrderItems struct {
	GUID                   string          `json:"guid,omitempty"`
	StockID                string          `json:"stock_id"`
	ProductAndServiceID    string          `json:"product_and_service_id"`
	ItemCode               string          `json:"ItemCode"`
	WarehouseData          []WarehouseData `json:"warehouse_data,omitempty"`
	UnitPrice              float64         `json:"UnitPrice"`
	TotalSum               float64         `json:"total_sum"`
	PaymentType            string          `json:"payment_type"`
	TotalSumBeforeDiscount float64         `json:"total_sum_before_discount"`
	SoldPrice              float64         `json:"sold_price"`
	WarehouseCode          string          `json:"WarehouseCode,omitempty"`
}
type WarehouseData struct {
	WarehouseGUID string `json:"warehouse_guid"`
	WarehouseCode string `json:"WarehouseCode,omitempty"`
	Quantity      string `json:"Quantity"`
}

// type Order struct {
// 	Data OrderData `json:"data"`
// }

// type OrderData struct {
// 	ClientID               string      `json:"client_id"`
// 	CardCode               string      `json:"CardCode"`
// 	DocDueDate             string      `json:"DocDueDate"`
// 	CreatedDate            string      `json:"DocDate"`
// 	UDirection             string      `json:"U_direction"`
// 	DirectionId            string      `json:"direction_id"`
// 	UDep                   string      `json:"U_dep"`
// 	SubdivisionID          string      `json:"subdivision_id"`
// 	Currency               string      `json:"currency"`
// 	TotalSumBeforeDiscount float64     `json:"total_sum_before_discount"` //float64
// 	Discount               float64     `json:"discount"`                  //float64
// 	Status                 []string    `json:"status"`
// 	CurrencyID             string      `json:"currency_id"`
// 	OrderItems             []OrderItem `json:"order_items"`
// 	DeliveryAddress        string      `json:"delivery_address"`
// 	WarehouseId            string      `json:"warehouse_id"`
// 	EmployeeID             string      `json:"employee_id"`
// }
// type OrderItem struct {
// 	StockGuid              string  `json:"stock_id"`
// 	ProductAndServiceID    string  `json:"product_and_service_id"`
// 	ItemCode               string  `json:"ItemCode"`
// 	Quantity               int     `json:"Quantity"` // int
// 	WarehouseCode          string  `json:"WarehouseCode"`
// 	WarehouseId            string  `json:"warehouse_id"`
// 	UnitPrice              float64 `json:"UnitPrice"` // float64
// 	TotalSum               float64 `json:"total_sum"` // float64
// 	PaymentType            string  `json:"payment_type"`
// 	TotalSumBeforeDiscount float64 `json:"total_sum_before_discount"` // float64
// 	DirectionId            string  `json:"direction_id"`
// 	DirectionName          string  `json:"direction_name"`
// 	SubdivisionName        string  `json:"subdivision_name"`
// }

type OrderSapItems struct {
	ItemCode string  `json:"ItemCode"`
	ItemName string  `json:"ItemName"`
	Quantity int     `json:"Quantity"`
	WhsCode  string  `json:"WhsCode"`
	Price    float64 `json:"price"`
}

type UpsertManyReqBody struct {
	Data UpsertManyData
}
type UpsertManyData struct {
	Objects   []map[string]interface{}
	FieldSlug string
}

type MultipleUpdateRequest struct {
	Data DataMultipleUpdate `json:"data"`
}

type DataMultipleUpdate struct {
	Objects []map[string]interface{} `json:"objects"`
}

type ReturnData struct {
	Code         string       `json:"code"`
	SaleID       string       `json:"sale_id"`
	ClientID     string       `json:"client_id"`
	ReturnDate   string       `json:"return_date"`
	CardCode     string       `json:"CardCode"`
	ClientName   string       `json:"client_name"`
	ReturnReason string       `json:"return_reason"`
	ReturnItems  []ReturnItem `json:"return_items"`
}

type ReturnItem struct {
	Guid                string `json:"guid"`
	ReturnQuantity      string `json:"return_quantity"`
	StockID             string `json:"stock_id"`
	ProductAndServiceID string `json:"product_and_service_id"`
	ItemCode            string `json:"ItemCode"`
	Quantity            string `json:"quantity"`
	WarehouseCode       string `json:"WarehouseCode"`
	UnitPrice           string `json:"UnitPrice"`
	WarehouseID         string `json:"warehouse_id"`
}

// movement models ...
type MovementRequest struct {
	GUID              string          `json:"guid"`
	DirectionID       string          `json:"direction_id"`
	DirectionName     string          `json:"direction_name"`
	SubdivisionID     string          `json:"subdivision_id"`
	SubdivisionName   string          `json:"subdivision_name"`
	SubdivisionID2    string          `json:"subdivision_id_2"`
	SubdivisionName2  string          `json:"subdivision_name_2"`
	EmployeeID        string          `json:"employee_id"`
	ClientID          string          `json:"client_id"`
	FromWarehouseID   string          `json:"from_warehouse_id"`
	FromWarehouseCode int             `json:"from_warehouse_code"`
	ToWarehouseID     string          `json:"to_warehouse_id"`
	ToWarehouseCode   int             `json:"to_warehouse_code"`
	CreatedDate       string          `json:"created_date"`
	MovementType      string          `json:"movement_type"`
	BaseEntry         int             `json:"base_entry"`
	Items             []MovementItems `json:"items"`
}
type MovementItems struct {
	ItemCode          int    `json:"item_code"`
	StockID           string `json:"stock_id"`
	Quantity          int    `json:"quantity"`
	FromWarehouseID   string `json:"from_warehouse_id"`
	FromWarehouseCode int    `json:"from_warehouse_code"`
	ToWarehouseID     string `json:"to_warehouse_id"`
	ToWarehouseCode   int    `json:"to_warehouse_code"`
}
