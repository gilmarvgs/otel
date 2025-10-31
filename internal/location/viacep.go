// Pacote location fornece funcionalidades para consulta de localização via API ViaCEP
package location

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	// Importação do OpenTelemetry para instrumentação HTTP
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// BaseURL é a URL base da API ViaCEP para consulta de CEP
// Formato: https://viacep.com.br/ws/{CEP}/json/
var BaseURL = "https://viacep.com.br/ws/%s/json/"

// Location representa a estrutura de resposta da API ViaCEP
type Location struct {
	City string `json:"localidade"` // Nome da cidade encontrada
}

// GetLocationByCEP consulta a API ViaCEP e retorna a localização com base no CEP.
// 
// IMPORTANTE: Esta função implementa rastreamento distribuído com OpenTelemetry:
// - Cria um span para medir o tempo de resposta da chamada à API ViaCEP
// - Usa cliente HTTP instrumentado para capturar métricas da requisição HTTP
// - Adiciona atributos ao span para facilitar debugging (CEP, URL, cidade, status)
//
// Parâmetros:
//   - ctx: Contexto com informações de rastreamento distribuído (spans)
//   - cep: CEP a ser consultado (deve ser string com 8 dígitos)
//
// Retorna:
//   - Location: Estrutura com o nome da cidade encontrada
//   - error: Erro caso a consulta falhe ou CEP não seja encontrado
func GetLocationByCEP(ctx context.Context, cep string) (Location, error) {
	// Obtém o tracer para criar spans de rastreamento
	tracer := otel.Tracer("location-service")
	
	// Cria um span para rastrear a chamada à API ViaCEP
	// Este span medirá o tempo total da requisição HTTP externa
	// Requisito: usar span para medir tempo de resposta do serviço de busca de CEP
	ctx, span := tracer.Start(ctx, "viacep-api-call")
	defer span.End() // Garante que o span será finalizado mesmo em caso de erro
	
	// Adiciona atributos ao span para facilitar análise e debugging
	// Esses atributos estarão disponíveis no Zipkin para visualização
	span.SetAttributes(
		attribute.String("viacep.cep", cep),              // CEP consultado
		attribute.String("http.url", fmt.Sprintf(BaseURL, cep)), // URL da requisição
	)
	
	url := fmt.Sprintf(BaseURL, cep)
	
	// Cria um cliente HTTP instrumentado com OpenTelemetry
	// O transporte OTEL automaticamente cria spans adicionais para a requisição HTTP
	// e captura métricas como latência, tamanho da requisição/resposta, etc.
	client := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	
	// Cria a requisição HTTP GET com contexto para propagação de traces
	// O contexto contém o span atual que será propagado através da rede
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		span.RecordError(err) // Registra o erro no span
		span.SetStatus(codes.Error, err.Error())
		return Location{}, err
	}
	
	// Executa a requisição HTTP à API ViaCEP
	// Esta é a chamada externa cujo tempo de resposta será medido pelo span
	resp, err := client.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return Location{}, err
	}
	defer resp.Body.Close() // Garante que o body será fechado
	
	// Adiciona o status HTTP ao span para indicar sucesso/falha da requisição
	span.SetAttributes(
		attribute.Int64("http.status_code", int64(resp.StatusCode)),
	)
	
	// Decodifica a resposta JSON da API ViaCEP
	var loc Location
	if err := json.NewDecoder(resp.Body).Decode(&loc); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return Location{}, err
	}

	// Valida se a cidade foi encontrada (resposta vazia indica CEP não encontrado)
	if loc.City == "" {
		err := fmt.Errorf("zipcode not found")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return Location{}, err
	}
	
	// Adiciona a cidade encontrada ao span para facilitar análise
	span.SetAttributes(
		attribute.String("viacep.city", loc.City),
	)
	// Marca o span como bem-sucedido
	span.SetStatus(codes.Ok, "CEP encontrado com sucesso")

	return loc, nil
}