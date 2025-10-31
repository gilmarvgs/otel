// Pacote principal do Serviço B
// Este serviço é responsável pela orquestração:
// 1. Recebe CEP válido do Serviço A
// 2. Consulta localização na API ViaCEP
// 3. Consulta temperatura na API WeatherAPI
// 4. Converte temperaturas (Celsius, Fahrenheit, Kelvin)
// 5. Retorna resposta formatada
package main

// Importação das dependências necessárias
import (
	"cep-weather/internal/location"  // Pacote para consulta de CEP via ViaCEP
	"cep-weather/internal/telemetry" // Pacote para configuração de telemetria OpenTelemetry
	"cep-weather/internal/weather"   // Pacote para consulta de temperatura via WeatherAPI
	"context"                        // Pacote para manipulação de contexto (rastreamento distribuído)
	"encoding/json"                  // Pacote para codificação/decodificação JSON
	"fmt"                            // Pacote para formatação e impressão
	"net/http"                       // Pacote para servidor HTTP
	"os"                             // Pacote para interação com o sistema operacional (variáveis de ambiente)
	"regexp"                         // Pacote para expressões regulares (validação de CEP)

	// Pacotes do OpenTelemetry para rastreamento distribuído
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

// WeatherResponse define a estrutura da resposta JSON do serviço
// Formato de resposta conforme especificação dos requisitos
type WeatherResponse struct {
	City  string  `json:"city"`   // Nome da cidade encontrada via ViaCEP
	TempC float64 `json:"temp_C"` // Temperatura em Celsius (da WeatherAPI)
	TempF float64 `json:"temp_F"` // Temperatura em Fahrenheit (convertida: F = C * 1.8 + 32)
	TempK float64 `json:"temp_K"` // Temperatura em Kelvin (convertida: K = C + 273)
}

// handler é a função que processa as requisições HTTP recebidas do Serviço A
// Implementa toda a lógica de orquestração do Serviço B conforme requisitos:
// - Validação de CEP
// - Consulta à API ViaCEP (com span de rastreamento)
// - Consulta à API WeatherAPI (com span de rastreamento)
// - Conversão de temperaturas
// - Retorno de resposta formatada
func handler(w http.ResponseWriter, r *http.Request) {
	// Inicializa o tracer do OpenTelemetry para criar spans de rastreamento
	// O contexto do request já contém informações de rastreamento do Serviço A
	tracer := otel.Tracer("service-b")
	ctx := r.Context()
	ctx, span := tracer.Start(ctx, "process-weather-request")
	defer span.End() // Garante que o span será finalizado

	// Validação: Verifica se o método HTTP é POST (conforme requisito)
	if r.Method != http.MethodPost {
		span.RecordError(fmt.Errorf("método não permitido: %s", r.Method))
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Define a estrutura para receber o CEP do JSON
	var input struct {
		CEP string `json:"cep"`
	}

	// Decodifica o JSON do corpo da requisição
	// Se falhar, retorna erro 422 conforme especificação
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		span.RecordError(err)
		http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity) // 422 conforme requisito
		return
	}

	// Validação: Verifica se o CEP contém exatamente 8 dígitos numéricos
	// Requisito: CEP deve ser uma string válida com 8 dígitos
	if !regexp.MustCompile(`^\d{8}$`).MatchString(input.CEP) {
		span.RecordError(fmt.Errorf("formato de CEP inválido: %s", input.CEP))
		http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity) // 422 conforme requisito
		return
	}

	// Consulta a localização usando a API ViaCEP
	// IMPORTANTE: A função GetLocationByCEP cria um span interno para medir
	// o tempo de resposta da chamada externa à API ViaCEP
	loc, err := location.GetLocationByCEP(ctx, input.CEP)
	if err != nil {
		span.RecordError(err)
		// Requisito: Retorna 404 se CEP não for encontrado
		if err.Error() == "zipcode not found" {
			http.Error(w, "can not find zipcode", http.StatusNotFound) // 404 conforme requisito
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Consulta a temperatura usando a WeatherAPI
	// IMPORTANTE: A função GetTemperature cria um span interno para medir
	// o tempo de resposta da chamada externa à API WeatherAPI
	tempC, err := weather.GetTemperature(ctx, loc.City)
	if err != nil {
		span.RecordError(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Calcula as conversões de temperatura conforme fórmulas especificadas
	// Fahrenheit: F = C * 1.8 + 32
	tempF := tempC*1.8 + 32
	// Kelvin: K = C + 273
	tempK := tempC + 273

	// Monta a resposta no formato especificado nos requisitos
	// Requisito: HTTP 200 com JSON contendo city, temp_C, temp_F, temp_K
	resp := WeatherResponse{
		City:  loc.City,
		TempC: tempC,
		TempF: tempF,
		TempK: tempK,
	}

	// Define o cabeçalho e envia a resposta JSON ao cliente
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200 conforme requisito
	json.NewEncoder(w).Encode(resp)
}

// função principal - ponto de entrada da aplicação
func main() {
	// Inicializa o OpenTelemetry com o nome do serviço
	// Isso configura o sistema de rastreamento distribuído e conexão com Zipkin
	tp, err := telemetry.InitTracer("service-b")
	if err != nil {
		fmt.Printf("Erro ao inicializar o tracer: %v\n", err)
		os.Exit(1)
	}
	// Garante que o tracer será desligado corretamente ao encerrar a aplicação
	// Isso é importante para enviar todos os traces pendentes ao Zipkin
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			fmt.Printf("Erro ao desligar o provedor de traces: %v\n", err)
		}
	}()

	// Configura a porta do servidor HTTP
	// Permite configurar via variável de ambiente (útil para Docker)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081" // Porta padrão para o Serviço B conforme requisitos
	}

	// Configura o handler HTTP com instrumentação OpenTelemetry
	// O otelhttp.NewHandler automaticamente cria spans para cada requisição
	// e propaga o contexto de rastreamento distribuído
	handler := otelhttp.NewHandler(http.HandlerFunc(handler), "weather-handler")
	http.Handle("/weather", handler) // Endpoint: POST /weather

	// Inicia o servidor HTTP na porta configurada
	fmt.Printf("Serviço B rodando na porta %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Erro ao iniciar o servidor: %v\n", err)
		os.Exit(1)
	}
}
