// Pacote principal do Serviço A
// Este serviço é responsável por receber requisições POST com CEP, validá-las
// e encaminhá-las ao Serviço B para processamento
package main

// Importação das dependências necessárias
import (
	"bytes"
	"cep-weather/internal/telemetry"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"

	// Importações para OpenTelemetry - usado para rastreamento distribuído
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

// Request define a estrutura do payload JSON recebido do cliente
// Requisito: deve receber um objeto JSON com campo "cep" contendo 8 dígitos
type Request struct {
	CEP string `json:"cep"` // Campo CEP que será recebido no JSON (deve ser string com 8 dígitos)
}

// handler é a função que processa as requisições HTTP recebidas
// Esta função implementa a lógica principal do Serviço A:
// 1. Valida se é requisição POST
// 2. Valida formato do CEP
// 3. Cria spans de rastreamento
// 4. Encaminha requisição ao Serviço B
// 5. Repassa a resposta ao cliente
func handler(w http.ResponseWriter, r *http.Request) {
	// Validação: Verifica se o método HTTP é POST (conforme requisito)
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decodifica o JSON do corpo da requisição
	// Se falhar na decodificação, retorna erro 422 conforme especificação
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity) // 422 conforme requisito
		return
	}

	// Validação: Verifica se o CEP contém exatamente 8 dígitos numéricos (string)
	// Requisito: CEP deve ser uma string válida com 8 dígitos
	if !regexp.MustCompile(`^\d{8}$`).MatchString(req.CEP) {
		http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity) // 422 conforme requisito
		return
	}

	// Cria um span para rastreamento distribuído com OpenTelemetry
	// Este span será propagado para o Serviço B através do contexto HTTP
	tracer := otel.Tracer("service-a")
	ctx := r.Context()
	ctx, span := tracer.Start(ctx, "process-zipcode")
	defer span.End() // Garante que o span será finalizado mesmo em caso de erro

	// Obtém a URL do Serviço B das variáveis de ambiente
	// Permite configurar a URL dinamicamente (útil para Docker/containers)
	serviceBURL := os.Getenv("SERVICE_B_URL")
	if serviceBURL == "" {
		serviceBURL = "http://localhost:8081/weather" // Valor padrão para desenvolvimento local
	}

	// Converte a requisição para JSON para enviar ao Serviço B
	// O Serviço B também espera receber JSON no formato {"cep": "29902555"}
	jsonBody, err := json.Marshal(req)
	if err != nil {
		span.RecordError(err) // Registra o erro no span para rastreamento
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Cria um cliente HTTP instrumentado com OpenTelemetry
	// O transporte OTEL automaticamente cria spans para requisições HTTP
	// e propaga o contexto de rastreamento distribuído
	client := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	
	// Cria a requisição HTTP POST com contexto para propagação de traces
	// O contexto contém informações de rastreamento que serão propagadas ao Serviço B
	httpReq, err := http.NewRequestWithContext(ctx, "POST", serviceBURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		span.RecordError(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Envia a requisição para o Serviço B
	// Esta chamada será automaticamente rastreada pelo OpenTelemetry
	resp, err := client.Do(httpReq)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close() // Garante que o body será fechado

	// Repassa o código de status e cabeçalhos da resposta do Serviço B
	// O Serviço A funciona como um proxy, repassando a resposta ao cliente
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)

	// Copia o corpo da resposta do Serviço B para o cliente
	// Esta é uma operação eficiente que evita carregar todo o body na memória
	if _, err := io.Copy(w, resp.Body); err != nil {
		span.RecordError(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// função principal - ponto de entrada da aplicação
func main() {
	// Inicializa o OpenTelemetry com o nome do serviço
	// Isso configura o sistema de rastreamento distribuído e conexão com Zipkin
	tp, err := telemetry.InitTracer("service-a")
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
		port = "8080" // Porta padrão para o Serviço A conforme requisitos
	}

	// Configura o handler HTTP com instrumentação OpenTelemetry
	// O otelhttp.NewHandler automaticamente cria spans para cada requisição
	handler := otelhttp.NewHandler(http.HandlerFunc(handler), "weather-handler")
	http.Handle("/weather", handler) // Endpoint: POST /weather

	// Inicia o servidor HTTP na porta configurada
	fmt.Printf("Serviço A rodando na porta %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Erro ao iniciar o servidor: %v\n", err)
		os.Exit(1)
	}
}
