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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

	cron, err := regCronjobs(handler)
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
		fmt.Println("CANCEL CAME HERE")
		cancel()
	}()

	// Use a gorilla mux for handling all HTTP requests
	r := chi.NewRouter()
	r.Use(middleware.Logger)    // log every requests
	r.Use(middleware.Recoverer) // handle panics

	// Add handlers for readiness and liveness endpoints
	r.HandleFunc("/health/{endpoint:readiness|liveness}", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/orders", handler.OrderCreate)
		// router.Put("/orders", handler.Order())
		// router.Delete("/orders", handler.Order())

		r.Get("/test", handler.JustTest)

		r.Post("/conversion", nil)
	})

	srv := &http.Server{
		Addr:           fmt.Sprintf(":%d", *port),
		Handler:        r,
		ReadTimeout:    1 * time.Minute,
		WriteTimeout:   1 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	listenAndServeErr := make(chan error, 1)
	go func() {
		// if *verbose {
		params.Log.Info().Msgf("Listening on :%d", *port)
		// }
		err := srv.ListenAndServe()
		// cancel()
		listenAndServeErr <- err
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancelFn := context.WithTimeout(context.Background(), time.Second*5)
	defer shutdownCancelFn()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "error on server shutdown: %v\n", err)
	}

	err = <-listenAndServeErr
	if http.ErrServerClosed == err {
		return nil
	}
	return err
}

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
func regCronjobs(h *function.Handler) (*cron.Cron, error) {
	var c = cron.New()

	// Scheduled the task to run at every 5 minutes
	_, err := c.AddFunc("*/5 * * * *", func() {
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

	// Scheduled the task to run at 5:00 AM every day
	_, err = c.AddFunc("0 5 * * *", func() {
		fmt.Println("Cronjob create or update whs task running at", time.Now())
		h.CreateOrUpdateWarehouse()
	})
	if err != nil {
		fmt.Println("Error scheduling task:", err)
	}

	return c, nil

}
