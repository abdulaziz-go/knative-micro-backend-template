package internal

import (
	"encoding/json"
	"fmt"
	"function/pkg"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/robfig/cron/v3"
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

func (h *Handler) HandleError(w http.ResponseWriter, err error, msg string) {
	h.Log.Err(err).Msg(msg + err.Error())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(`{"error": "` + err.Error() + `"}`))
}

func (h *Handler) JustTest(w http.ResponseWriter, r *http.Request) {
	// bytes, err := io.ReadAll(r.Body)
	// if err != nil {
	// 	h.Log.Err(err).Msg("Error on reading request body on Just Test ")
	// 	return
	// }

	// h.Log.Info().Msg(string(bytes))
	// h.ExchangeRate()
	if err := h.UpdateStock(); err != nil {
		h.Log.Err(err).Msg("Error on updating stock")
	}
	h.Log.Info().Msg("Stock function successfully")
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

// register all cronjobs here...
func (h *Handler) RegCronjobs() (*cron.Cron, error) {
	var c = cron.New()

	// Scheduled the task to run at every 5 minutes
	_, err := c.AddFunc("*/5 * * * *", func() {
		fmt.Println("Cronjob exchange task running at", time.Now().Format(time.RFC3339))
		h.ExchangeRate()
	})
	if err != nil {
		h.Log.Err(err).Msg("Error scheduling task, ExchangeRate")
		return nil, err

	}

	// Scheduled the task to run at 9:00 AM every day
	_, err = c.AddFunc("0 9 * * *", func() {
		fmt.Println("Cronjob ItemGroupCronjob task running at", time.Now().Format(time.RFC3339))
		h.CreateItemGroup()
	})
	if err != nil {
		h.Log.Err(err).Msg("Error scheduling task, CreateItemGroup")
		return nil, err
	}

	// Scheduled the task to run every hour minutes 0
	_, err = c.AddFunc("0 * * * *", func() {
		fmt.Println("Cronjob ProductAndServiceCronJob task running at", time.Now().Format(time.RFC3339))
		h.ProductAndServiceCronJob()
	})
	if err != nil {
		h.Log.Err(err).Msg("Error scheduling task, ProductAndServiceCronJob")
		return nil, err
	}

	// Scheduled the task to run every hour minutes 5
	_, err = c.AddFunc("5 * * * *", func() {
		fmt.Println("Cronejob stock running at", time.Now().Format(time.RFC3339))
		h.UpdateStock()
	})
	if err != nil {
		h.Log.Err(err).Msg("Error scheduling task, UpdateStock")
		return nil, err
	}

	// Scheduled the task to run at 5:00 AM every day
	_, err = c.AddFunc("0 5 * * *", func() {
		fmt.Println("Cronjob create or update whs task running at", time.Now().Format(time.RFC3339))
		h.CreateOrUpdateWarehouse()
	})
	if err != nil {
		h.Log.Err(err).Msg("Error scheduling task, CreateOrUpdateWarehouse")
		return nil, err
	}

	return c, nil
}
