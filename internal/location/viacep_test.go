package location

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetLocationByCEP(t *testing.T) {
	// Fake response simulating ViaCEP
	fakeResponse := `{"localidade": "TesteCity"}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fakeResponse)
	}))
	defer ts.Close()

	// Substitui temporariamente a URL base
	originalBaseURL := BaseURL
	BaseURL = ts.URL + "/%s/json"
	defer func() { BaseURL = originalBaseURL }()

	ctx := context.Background()
	loc, err := GetLocationByCEP(ctx, "12345678")
	if err != nil {
		t.Errorf("Erro inesperado: %v", err)
	}

	if loc.City != "TesteCity" {
		t.Errorf("Esperado %s, obteve %s", "TesteCity", loc.City)
	}
}
