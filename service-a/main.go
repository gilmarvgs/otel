// Pacote principal do Serviço A
package main

// Importação das dependências necessárias
import (
	"bytes"
	"cep-weather/internal/telemetry"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"

	// Importações para OpenTelemetry
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

// Request define a estrutura do payload JSON recebido
type Request struct {
	CEP string `json:"cep"` // Campo CEP que será recebido no JSON
}

// handler é a função que processa as requisições HTTP
func handler(w http.ResponseWriter, r *http.Request) {
	// Verifica se o método HTTP é POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decodifica o JSON do corpo da requisição
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity)
		return
	}

	// Valida o formato do CEP (8 dígitos)
	if !regexp.MustCompile(`^\d{8}$`).MatchString(req.CEP) {
		http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity)
		return
	}

	// Cria um span para rastreamento com OpenTelemetry
	tracer := otel.Tracer("service-a")
	ctx := r.Context()
	ctx, span := tracer.Start(ctx, "process-zipcode")
	defer span.End()

	// Obtém a URL do Serviço B das variáveis de ambiente
	serviceBURL := os.Getenv("SERVICE_B_URL")
	if serviceBURL == "" {
		serviceBURL = "http://localhost:8081/weather"
	}

	// Converte a requisição para JSON
	jsonBody, err := json.Marshal(req)
	if err != nil {
		span.RecordError(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Cria um cliente HTTP instrumentado com OpenTelemetry
	client := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", serviceBURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		span.RecordError(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Envia a requisição para o Serviço B
	resp, err := client.Do(httpReq)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Repassa o código de status e cabeçalhos da resposta do Serviço B
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)

	// Copia o corpo da resposta do Serviço B
	if _, err := fmt.Fprintf(w, "%s", resp.Body); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// função principal
func main() {
	// Inicializa o OpenTelemetry
	tp, err := telemetry.InitTracer("service-a")
	if err != nil {
		fmt.Printf("Erro ao inicializar o tracer: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			fmt.Printf("Erro ao desligar o provedor de traces: %v\n", err)
		}
	}()

	// Configura a porta do servidor
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Usa a porta 8080 para o Serviço A
	}

	// Configura o handler HTTP com OpenTelemetry
	handler := otelhttp.NewHandler(http.HandlerFunc(handler), "weather-handler")
	http.Handle("/weather", handler)

	// Inicia o servidor HTTP
	fmt.Printf("Serviço A rodando na porta %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Erro ao iniciar o servidor: %v\n", err)
		os.Exit(1)
	}
}
