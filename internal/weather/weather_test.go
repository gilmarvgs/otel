package weather

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGetTemperature_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"current":{"temp_c":21.5}}`)
	}))
	defer srv.Close()

	origApiURL := ApiURL
	ApiURL = srv.URL + "/?key=%s&q=%s"
	defer func() { ApiURL = origApiURL }()

	origKey := os.Getenv("WEATHER_API_KEY")
	os.Setenv("WEATHER_API_KEY", "testkey")
	defer os.Setenv("WEATHER_API_KEY", origKey)

	temp, err := GetTemperature("Sao Paulo")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if temp != 21.5 {
		t.Fatalf("expected 21.5, got %v", temp)
	}
}

func TestGetTemperature_ApiError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"error":{"message":"invalid api key"}}`)
	}))
	defer srv.Close()

	origApiURL := ApiURL
	ApiURL = srv.URL + "/?key=%s&q=%s"
	defer func() { ApiURL = origApiURL }()

	origKey := os.Getenv("WEATHER_API_KEY")
	os.Setenv("WEATHER_API_KEY", "badkey")
	defer os.Setenv("WEATHER_API_KEY", origKey)

	_, err := GetTemperature("Sao Paulo")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestGetTemperature_NoApiKey(t *testing.T) {
	origKey := os.Getenv("WEATHER_API_KEY")
	os.Unsetenv("WEATHER_API_KEY")
	defer func() {
		if origKey != "" {
			os.Setenv("WEATHER_API_KEY", origKey)
		}
	}()

	origApiURL := ApiURL
	ApiURL = "https://example.invalid/?key=%s&q=%s"
	defer func() { ApiURL = origApiURL }()

	_, err := GetTemperature("Sao Paulo")
	if err == nil || err.Error() != "WEATHER_API_KEY not set" {
		t.Fatalf("expected WEATHER_API_KEY not set error, got %v", err)
	}
}
