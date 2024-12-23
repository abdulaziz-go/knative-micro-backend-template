package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"function/pkg"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	// cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/go-chi/chi/v5"
	"github.com/robfig/cron/v3"

	function "function/internal"
)

// Here is Azizbek's implementation of recoverMiddleware:

var (
	usage   = "run\n\nRuns a CloudFunction CloudEvent handler."
	verbose = flag.Bool("V", false, "Verbose logging [$VERBOSE]")
	port    = flag.Int("port", 8080, "Listen on all interfaces at the given port [$PORT]")
)

func main() {

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, usage)
		flag.PrintDefaults()
	}
	parseEnv()   // override static defaults with environment.
	flag.Parse() // override env vars with flags.

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run a cloudevents client in receive mode which invokes
// the user-defined function.Handler on receipt of an event.
func run() error {
	// cronjob
	var (
		cfg, _  = pkg.NewConfig()
		params  = pkg.NewParams(cfg)
		handler = function.InitHandler(params)
	)

	cron, err := regCronjob(handler)
	if err != nil {
		handler.Log.Err(err).Msg("Error on cronjob")
	}

	cron.Start()
	defer cron.Stop()

	ctx, cancel := context.WithCancel(context.Background())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	// Use a gorilla mux for handling all HTTP requests
	router := chi.NewRouter()

	// Add handlers for readiness and liveness endpoints
	router.HandleFunc("/health/{endpoint:readiness|liveness}", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})

	/*
		var httpHandler = toHttpHandler(handler, ctx)

		if httpHandler == nil {
			if *verbose {
				fmt.Printf("Initializing CloudEvent function\n")
			}
			protocol, err := cloudevents.NewHTTP(
				cloudevents.WithPort(*port),
				cloudevents.WithPath("/"),
			)
			if err != nil {
				fmt.Println("Error on creating protocol:", err)
				return err
			}
			eventHandler, err := cloudevents.NewHTTPReceiveHandler(ctx, protocol, handler)
			if err != nil {
				fmt.Println("Error on creating proto1col:", err)

				return err
			}
			router.Handle("/", eventHandler)
		} else {
			if *verbose {
				fmt.Printf("Initializing HTTP function\n")
			}
			router.Handle("/", httpHandler)
		}
	*/
	router.Group(func(r chi.Router) {

		router.Post("/orders", handler.Order)
		// router.Get("/orders", handler.Order())
		// router.Put("/orders", handler.Order())
		// router.Delete("/orders", handler.Order())
	})

	httpServer := &http.Server{
		Addr:           fmt.Sprintf(":%d", *port),
		Handler:        router,
		ReadTimeout:    1 * time.Minute,
		WriteTimeout:   1 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	listenAndServeErr := make(chan error, 1)
	go func() {
		if *verbose {
			params.Log.Info().Msgf("Listening on :%d", *port)
		}
		err := httpServer.ListenAndServe()
		cancel()
		listenAndServeErr <- err
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancelFn := context.WithTimeout(context.Background(), time.Second*5)
	defer shutdownCancelFn()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "error on server shutdown: %v\n", err)
	}

	err = <-listenAndServeErr
	if http.ErrServerClosed == err {
		return nil
	}
	return err
}

// if the handler signature is compatible with http handler the function returns an instance of `http.Handler`,
// otherwise nil is returned
// func toHttpHandler(handler interface{}, ctx context.Context) http.Handler {

// 	if f, ok := handler.(func(rw http.ResponseWriter, req *http.Request)); ok {
// 		return recoverMiddleware(http.HandlerFunc(f))
// 	}

// 	if f, ok := handler.(func(ctx context.Context, rw http.ResponseWriter, req *http.Request)); ok {
// 		ff := func(rw http.ResponseWriter, req *http.Request) {
// 			f(ctx, rw, req)
// 		}
// 		return recoverMiddleware(http.HandlerFunc(ff))
// 	}

// 	return nil
// }

// parseEnv parses environment variables, populating the destination flags
// prior to the builtin flag parsing.  Invalid values exit 1.
func parseEnv() {
	parseBool := func(key string, dest *bool) {
		if val, ok := os.LookupEnv(key); ok {
			b, err := strconv.ParseBool(val)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v is not a valid boolean\n", key)
				os.Exit(1)
			}
			*dest = b
		}
	}
	parseInt := func(key string, dest *int) {
		if val, ok := os.LookupEnv(key); ok {
			n, err := strconv.Atoi(val)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v is not a valid integer\n", key)
				os.Exit(1)
			}
			*dest = n
		}
	}

	parseBool("VERBOSE", verbose)
	parseInt("PORT", port)
}

// register all cronjobs here...
func regCronjob(h *function.Handler) (*cron.Cron, error) {
	var c = cron.New()

	// Scheduled the task to run at 5:00 AM every day
	_, err := c.AddFunc("0 5 * * *", func() {
		fmt.Println("Cronjob exchange task running at", time.Now())
		h.ExchangeRate()
	})
	if err != nil {
		fmt.Println("Error scheduling task:", err)
	}
	// Scheduled the task to run at 9:00 AM every day
	_, err = c.AddFunc("0 9 * * *", func() {
		fmt.Println("Cronjob ItemGroupCronjob task running at", time.Now())
		h.ItemGroupCronjob()
	})
	if err != nil {
		fmt.Println("Error scheduling task:", err)
	}

	// Scheduled the task to run every hour minutes 0
	_, err = c.AddFunc("0 * * * *", func() {
		fmt.Println("Cronjob ProductAndServiceCronJob task running at", time.Now())
		h.ProductAndServiceCronJob()
	})
	if err != nil {
		fmt.Println("Error scheduling task:", err)
	}

	// Scheduled the task to run every hour minutes 5
	_, err = c.AddFunc("5 * * * *", func() {
		fmt.Println("Cronejob stock running at", time.Now())
		h.StockCronJob()
	})
	if err != nil {
		fmt.Println("Error scheduling task:", err)
	}

	return c, nil

}
