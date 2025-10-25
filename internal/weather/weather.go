package weather

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

// ApiURL can be overridden for testing purposes
var ApiURL = "https://api.weatherapi.com/v1/current.json?key=%s&q=%s"

// WeatherResponse represents the response from WeatherAPI
// Example response:
// {
//   "current": {
//     "temp_c": 25.0,
//     ...
//   }
// }

type WeatherResponse struct {
	Current struct {
		TempC float64 `json:"temp_c"`
	} `json:"current"`
}

// GetTemperature queries the WeatherAPI to get the current temperature in Celsius for a given city.
func GetTemperature(city string) (float64, error) {
	apiKey := os.Getenv("WEATHER_API_KEY")
	if apiKey == "" {
		return 0, fmt.Errorf("WEATHER_API_KEY not set")
	}
	//curl "https://api.weatherapi.com/v1/current.json?key=35324af72c7249b1bbc00833252409&q=Sao-Paulo"
	//curl "https://api.weatherapi.com/v1/current.json?key=35324af72c7249b1bbc00833252409&q=Sao%20Paulo"

	escapedCity := url.QueryEscape(city)
	fullURL := fmt.Sprintf(ApiURL, apiKey, escapedCity)

	fmt.Println("Consultando WeatherAPI para cidade:", city)
	fmt.Println("URL:", fullURL)

	resp, err := http.Get(fullURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Adicionando logs detalhados para diagnóstico
	if resp.StatusCode != http.StatusOK {
		// Lê o corpo da resposta para depuração adicional
		body, errBody := io.ReadAll(resp.Body)
		if errBody != nil {
			log.Printf("Erro ao ler o corpo da resposta: %v", errBody)
		} else {
			log.Printf("Falha na consulta do clima: status %d, resposta: %s", resp.StatusCode, string(body))
		}
		return 0, fmt.Errorf("weather lookup failed")
	}

	var weatherResp WeatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&weatherResp); err != nil {
		return 0, err
	}

	return weatherResp.Current.TempC, nil
}
