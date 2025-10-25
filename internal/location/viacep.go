package location

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// BaseURL é a URL base da API ViaCEP.
var BaseURL = "https://viacep.com.br/ws/%s/json/"

type Location struct {
	City string `json:"localidade"`
}

// GetLocationByCEP consulta a API ViaCEP e retorna a localização com base no CEP.
func GetLocationByCEP(cep string) (Location, error) {
	url := fmt.Sprintf(BaseURL, cep)
	resp, err := http.Get(url)
	if err != nil {
		return Location{}, err
	}
	defer resp.Body.Close()

	var loc Location
	if err := json.NewDecoder(resp.Body).Decode(&loc); err != nil {
		return Location{}, err
	}

	if loc.City == "" {
		return Location{}, fmt.Errorf("zipcode not found")
	}

	return loc, nil
}
