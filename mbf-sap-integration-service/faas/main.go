package main

import (
	"context"

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
	go sendRequestEvery5Seconds()

	var (
		cfg, _  = pkg.NewConfig()
		params  = pkg.NewParams(cfg)
		handler = function.InitHandler(params)
	)

	cron, err := handler.RegCronjobs()
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
	r := chi.NewRouter()
	r.Use(middleware.Logger)    // log every requests
	r.Use(middleware.Recoverer) // handle panics

	// Add handlers for readiness and liveness endpoints
	r.HandleFunc("/health/{endpoint:readiness|liveness}", func(w http.ResponseWriter, r *http.Request) {
		handler.Log.Info().Msg("health api")
		w.WriteHeader(http.StatusOK)
		// json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/orders", handler.OrderCreate)
		// router.Put("/orders", handler.Order())
		// router.Delete("/orders", handler.Order())

		r.Post("/movement", handler.MovementRequest)
		// r.Post("/movement/send")
		// r.Post("/movement/")

		r.Get("/scanner", handler.Scanner)

		r.Get("/test", handler.JustTest)
		r.Post("/conversion", nil) // konvertatsiya
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
		params.Log.Info().Msgf("Listening on :%d", *port)
		err := srv.ListenAndServe()
		cancel()
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

func sendRequestEvery5Seconds() {
	url := pkg.KnativeURL + "health/liveness"

	for {
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Error sending request:", err)
			continue
		}

		resp.Body.Close()

		time.Sleep(5 * time.Second)
	}
}
