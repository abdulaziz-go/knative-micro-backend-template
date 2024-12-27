package internal

import (
	"encoding/json"
	"fmt"
	// "function/internal"
	"function/pkg"
	"io"
	"log"
	"net/http"

	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/mongo"
)

type Handler struct {
	Log     zerolog.Logger
	MongoDB *mongo.Database
}

func InitHandler(params *pkg.Params) *Handler {
	conn, err := pkg.MongoConn()
	if err != nil {
		log.Fatal("error while connecting to mongo", err)
	}

	return &Handler{
		Log:     params.Log,
		MongoDB: conn,
	}
}

func (h *Handler) JustTest(w http.ResponseWriter, r *http.Request) {
	// bytes, err := io.ReadAll(r.Body)
	// if err != nil {
	// 	h.Log.Err(err).Msg("Error on reading request body on Just Test ")
	// 	return
	// }

	// h.Log.Info().Msg(string(bytes))
	h.ExchangeRate()
	h.Log.Info().Msg("CreateOrUpdateWarehouse function successfully")
	response := map[string]interface{}{
		"message": "Order created successfully",
		"status":  200,
	}
	pkg.HandleResponse(w, response, http.StatusOK)
}

func (h *Handler) Return() http.HandlerFunc {
	if err := pkg.LoginSAP(); err != nil {
		h.Log.Err(err).Msg("Error on login SAP")
		return nil

	}
	return func(w http.ResponseWriter, r *http.Request) {
		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			h.Log.Err(err).Msg("Error on reading request body on Just Test ")
			return
		}

		h.Log.Info().Msg(string(bytes))

		w.Write([]byte("Just test api worked"))
	}

}

func (h *Handler) OrderCreate(w http.ResponseWriter, r *http.Request) {
	if err := pkg.LoginSAP(); err != nil {
		h.Log.Err(err).Msg("Error on login SAP")
		pkg.HandleResponse(w, err, http.StatusInternalServerError)
		return

	}

	requestByte, err := io.ReadAll(r.Body)
	if err != nil {
		h.Log.Err(err).Msg("Error on reading request body")
		pkg.HandleResponse(w, err, http.StatusBadRequest)
		return
	}

	// fmt.Println(string(requestByte))
	var orderRequest pkg.Order

	if err := json.Unmarshal(requestByte, &orderRequest); err != nil {
		h.Log.Err(err).Msg("Error on unmarshalling request body")
		pkg.HandleResponse(w, err, http.StatusBadRequest)
		return

	}

	// While creating a new orders, we need to create it in SAP B1 to
	if err := h.CreateOrder(&orderRequest); err != nil {
		h.Log.Err(err).Msg("Error on creating order in Ucode")
		pkg.HandleResponse(w, err, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message": "Order created successfully",
		"status":  200,
	}

	pkg.HandleResponse(w, response, http.StatusOK)

}

func (h *Handler) NewHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		requestByte, err := io.ReadAll(r.Body)
		// fmt.Println("REQUEST BODY: ", string(requestByte))
		if err != nil {
			h.Log.Err(err).Msg("Error on getting request body")
			pkg.HandleResponse(w, err, http.StatusBadRequest)
		}
		fmt.Println("HTTP button request body: ", string(requestByte))

		if err := pkg.LoginSAP(); err != nil {
			h.Log.Err(err).Msg("Error on login SAP")
			pkg.HandleResponse(w, err, http.StatusInternalServerError)
			return
		}

		pkg.HandleResponse(w, "ok", http.StatusOK)
	}
}

func (h *Handler) StockCronJob() {
	if err := pkg.LoginSAP(); err != nil {
		log.Fatal("error while login SAP", err)
	}

	if err := h.UpdateStock(); err != nil {
		log.Fatal("error while creating stock ", err)
	}

}

func (h *Handler) ItemGroupCronjob() {
	if err := pkg.LoginSAP(); err != nil {
		h.Log.Err(err).Msg("Error on login SAP ItemGroupCronjob")
		// log.Fatal("error while login SAP", err)
	}

	if err := h.CreateItemGroup(); err != nil {
		h.Log.Err(err).Msg("Error on creating item group cronjob")
	}
}

func (h *Handler) ProductsCronJob() {
	if err := pkg.LoginSAP(); err != nil {
		h.Log.Err(err).Msg("Error on login SAP ProductsCronJob")
	}

	if err := h.CreateProduct(); err != nil {
		h.Log.Err(err).Msg("Error on creating item group cronjob")
	}
}

func (h *Handler) ProductAndServiceCronJob() {
	if err := pkg.LoginSAP(); err != nil {
		h.Log.Err(err).Msg("Error on login SAP ProductAndServiceCronJob")
	}

	if err := h.CreateProductAndServices(); err != nil {
		h.Log.Err(err).Msg("Error on login SAP ProductAndServiceCronJob")
	}
}

// func (h *Handler) ExchangeRate() {

// 	if err := pkg.LoginSAP(); err != nil {
// 		log.Fatal("error while login SAP", err)
// 	}
// 	if err := h.CreateExchangeRate(); err != nil {
// 		log.Fatal("error while creating exchange rate ", err)
// 	}
// }
