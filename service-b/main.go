// Pacote principal do Serviço B
package main

// Importação das dependências necessárias
import (
	"cep-weather/internal/location"  // Pacote para consulta de CEP
	"cep-weather/internal/telemetry" // Pacote para telemetria
	"cep-weather/internal/weather"   // Pacote para consulta de clima
	"context"                        // Pacote para manipulação de contexto
	"encoding/json"                  // Pacote para codificação/decodificação JSON
	"fmt"                            // Pacote para formatação e impressão
	"net/http"                       // Pacote para servidor HTTP
	"os"                             // Pacote para interação com o sistema operacional
	"regexp"                         // Pacote para expressões regulares

	// Pacotes do OpenTelemetry
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

// WeatherResponse define a estrutura da resposta do serviço
type WeatherResponse struct {
	City  string  `json:"city"`   // Nome da cidade
	TempC float64 `json:"temp_C"` // Temperatura em Celsius
	TempF float64 `json:"temp_F"` // Temperatura em Fahrenheit
	TempK float64 `json:"temp_K"` // Temperatura em Kelvin
}

// handler é a função que processa as requisições HTTP
func handler(w http.ResponseWriter, r *http.Request) {
	// Inicializa o tracer do OpenTelemetry
	tracer := otel.Tracer("service-b")
	ctx := r.Context()
	ctx, span := tracer.Start(ctx, "process-weather-request")
	defer span.End()

	// Verifica se o método HTTP é POST
	if r.Method != http.MethodPost {
		span.RecordError(fmt.Errorf("método não permitido: %s", r.Method))
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Define a estrutura para receber o CEP
	var input struct {
		CEP string `json:"cep"`
	}

	// Decodifica o JSON do corpo da requisição
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		span.RecordError(err)
		http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity)
		return
	}

	// Valida o formato do CEP (8 dígitos)
	if !regexp.MustCompile(`^\d{8}$`).MatchString(input.CEP) {
		span.RecordError(fmt.Errorf("formato de CEP inválido: %s", input.CEP))
		http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity)
		return
	}

	// Consulta a localização usando a API ViaCEP
	ctx, locationSpan := tracer.Start(ctx, "get-location")
	loc, err := location.GetLocationByCEP(input.CEP)
	locationSpan.End()
	if err != nil {
		span.RecordError(err)
		if err.Error() == "zipcode not found" {
			http.Error(w, "can not find zipcode", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Consulta a temperatura usando a WeatherAPI
	ctx, weatherSpan := tracer.Start(ctx, "get-weather")
	tempC, err := weather.GetTemperature(loc.City)
	weatherSpan.End()
	if err != nil {
		span.RecordError(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Calcula as conversões de temperatura
	tempF := tempC*1.8 + 32 // Conversão para Fahrenheit
	tempK := tempC + 273    // Conversão para Kelvin

	// Monta a resposta
	resp := WeatherResponse{
		City:  loc.City,
		TempC: tempC,
		TempF: tempF,
		TempK: tempK,
	}

	// Define o cabeçalho e envia a resposta
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// função principal
func main() {
	// Inicializa o OpenTelemetry
	tp, err := telemetry.InitTracer("service-b")
	if err != nil {
		fmt.Printf("Erro ao inicializar o tracer: %v\n", err)
		os.Exit(1)
	}
	// Garante que o tracer será desligado corretamente ao finalizar
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			fmt.Printf("Erro ao desligar o provedor de traces: %v\n", err)
		}
	}()

	// Configura a porta do servidor
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081" // Usa a porta 8081 para o Serviço B
	}

	// Configura o handler HTTP com OpenTelemetry
	handler := otelhttp.NewHandler(http.HandlerFunc(handler), "weather-handler")
	http.Handle("/weather", handler)

	// Inicia o servidor HTTP
	fmt.Printf("Serviço B rodando na porta %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Erro ao iniciar o servidor: %v\n", err)
		os.Exit(1)
	}
}
