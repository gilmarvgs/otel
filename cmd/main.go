package main

import (
	"cep-weather/internal/location"
	"cep-weather/internal/weather"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
)

type Response struct {
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	cep := r.URL.Query().Get("cep")
	if !regexp.MustCompile(`^\d{8}$`).MatchString(cep) {
		http.Error(w, "CEP invalido", http.StatusUnprocessableEntity)
		return
	}

	loc, err := location.GetLocationByCEP(cep)
	if err != nil {
		http.Error(w, "CEP nao encontrado", http.StatusNotFound)
		return
	}

	tempC, err := weather.GetTemperature(loc.City)
	if err != nil {
		http.Error(w, "Falha ao obter temperatura", http.StatusInternalServerError)
		return
	}

	resp := Response{
		TempC: tempC,
		TempF: tempC*1.8 + 32,
		TempK: tempC + 273,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	http.HandleFunc("/", handler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Listening on port", port)
	http.ListenAndServe(":"+port, nil)
}
