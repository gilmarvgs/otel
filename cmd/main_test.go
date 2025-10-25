package main

import (
	"cep-weather/internal/location"
	"cep-weather/internal/weather"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHandlerInvalidCEP(t *testing.T) {
	req := httptest.NewRequest("GET", "/?cep=123", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d", http.StatusUnprocessableEntity, w.Code)
	}
}

func TestHandlerNotFoundCEP(t *testing.T) {
	// Para simular um CEP não encontrado, sobrescreve temporariamente a BaseURL para retornar resposta vazia
	originalBaseURL := location.BaseURL
	location.BaseURL = "http://example.com/invalid/%s/json"
	defer func() { location.BaseURL = originalBaseURL }()

	req := httptest.NewRequest("GET", "/?cep=00000000", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandlerSuccess(t *testing.T) {
	// Fake servidor para ViaCEP
	locationFakeResponse := `{"localidade": "TestCity"}`
	locationTs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(locationFakeResponse))
	}))
	defer locationTs.Close()
	originalBaseURL := location.BaseURL
	location.BaseURL = locationTs.URL + "/%s/json"
	defer func() { location.BaseURL = originalBaseURL }()

	// Fake servidor para WeatherAPI
	weatherFakeResponse := `{"current": {"temp_c": 25.0}}`
	weatherTs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(weatherFakeResponse))
	}))
	defer weatherTs.Close()
	originalApiURL := weather.ApiURL
	// ApiURL expects two format verbs: one for the api key and one for the query (city)
	weather.ApiURL = weatherTs.URL + "/?key=%s&q=%s&aqi=no"
	defer func() { weather.ApiURL = originalApiURL }()

	// Define chave dummy
	os.Setenv("WEATHER_API_KEY", "dummy")
	defer os.Unsetenv("WEATHER_API_KEY")

	req := httptest.NewRequest("GET", "/?cep=12345678", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp Response
	err := json.NewDecoder(w.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.TempC != 25.0 {
		t.Errorf("Expected TempC 25.0, got %v", resp.TempC)
	}

	expectedF := 25.0*1.8 + 32
	if resp.TempF != expectedF {
		t.Errorf("Expected TempF %v, got %v", expectedF, resp.TempF)
	}

	expectedK := 25.0 + 273
	if resp.TempK != expectedK {
		t.Errorf("Expected TempK %v, got %v", expectedK, resp.TempK)
	}
}
