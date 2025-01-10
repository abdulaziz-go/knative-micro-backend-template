package function

import (
	"function/internal"
	"function/pkg"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandle ensures that Handle executes without error and returns the
// HTTP 200 status code indicating no errors.
func TestHandle(t *testing.T) {
	var (
		w = httptest.NewRecorder()
		// req = httptest.NewRequest("POST", "/test", bytes.NewBuffer([]byte(jsonData)))
		res *http.Response
	)

	cfg, _ := pkg.NewConfig()
	params := pkg.NewParams(cfg)
	handler := internal.InitHandler(params)

	handler.ProductAndServiceCronJob()

	res = w.Result()
	defer res.Body.Close()

	if res.StatusCode != 200 {
		t.Fatalf("unexpected response code: %v", res.StatusCode)
	}
}
