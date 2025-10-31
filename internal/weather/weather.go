// Pacote weather fornece funcionalidades para consulta de temperatura via API WeatherAPI
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	// Importação do OpenTelemetry para instrumentação HTTP
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ApiURL é a URL base da API WeatherAPI para consulta de temperatura
// Pode ser sobrescrita para fins de teste
// Formato: https://api.weatherapi.com/v1/current.json?key={API_KEY}&q={CITY}
var ApiURL = "https://api.weatherapi.com/v1/current.json?key=%s&q=%s"

// WeatherResponse representa a estrutura de resposta da API WeatherAPI
// Exemplo de resposta:
// {
//   "current": {
//     "temp_c": 25.0,  // Temperatura em Celsius
//     ...
//   }
// }
type WeatherResponse struct {
	Current struct {
		TempC float64 `json:"temp_c"` // Temperatura atual em graus Celsius
	} `json:"current"`
}

// GetTemperature consulta a WeatherAPI para obter a temperatura atual em Celsius para uma cidade.
//
// IMPORTANTE: Esta função implementa rastreamento distribuído com OpenTelemetry:
// - Cria um span para medir o tempo de resposta da chamada à API WeatherAPI
// - Usa cliente HTTP instrumentado para capturar métricas da requisição HTTP
// - Adiciona atributos ao span para facilitar debugging (cidade, URL, temperatura, status)
//
// Parâmetros:
//   - ctx: Contexto com informações de rastreamento distribuído (spans)
//   - city: Nome da cidade para consultar a temperatura
//
// Retorna:
//   - float64: Temperatura em graus Celsius
//   - error: Erro caso a consulta falhe (API key ausente, falha na requisição, etc.)
func GetTemperature(ctx context.Context, city string) (float64, error) {
	// Obtém o tracer para criar spans de rastreamento
	tracer := otel.Tracer("weather-service")
	
	// Cria um span para rastrear a chamada à API WeatherAPI
	// Este span medirá o tempo total da requisição HTTP externa
	// Requisito: usar span para medir tempo de resposta do serviço de busca de temperatura
	ctx, span := tracer.Start(ctx, "weatherapi-call")
	defer span.End() // Garante que o span será finalizado mesmo em caso de erro
	
	// Obtém a chave da API WeatherAPI das variáveis de ambiente
	// Esta chave é obrigatória e deve ser configurada antes da execução
	apiKey := os.Getenv("WEATHER_API_KEY")
	if apiKey == "" {
		err := fmt.Errorf("WEATHER_API_KEY not set")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}

	// Codifica o nome da cidade para URL (trata espaços e caracteres especiais)
	escapedCity := url.QueryEscape(city)
	fullURL := fmt.Sprintf(ApiURL, apiKey, escapedCity)

	// Adiciona atributos ao span para facilitar análise e debugging
	// Esses atributos estarão disponíveis no Zipkin para visualização
	span.SetAttributes(
		attribute.String("weatherapi.city", city),      // Cidade consultada
		attribute.String("http.url", fullURL),          // URL da requisição (sem API key por segurança)
	)

	fmt.Println("Consultando WeatherAPI para cidade:", city)
	fmt.Println("URL:", fullURL)

	// Cria um cliente HTTP instrumentado com OpenTelemetry
	// O transporte OTEL automaticamente cria spans adicionais para a requisição HTTP
	// e captura métricas como latência, tamanho da requisição/resposta, etc.
	client := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	
	// Cria a requisição HTTP GET com contexto para propagação de traces
	// O contexto contém o span atual que será propagado através da rede
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		span.RecordError(err) // Registra o erro no span
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}
	
	// Executa a requisição HTTP à API WeatherAPI
	// Esta é a chamada externa cujo tempo de resposta será medido pelo span
	resp, err := client.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}
	defer resp.Body.Close() // Garante que o body será fechado

	// Adiciona o status HTTP ao span para indicar sucesso/falha da requisição
	span.SetAttributes(
		attribute.Int64("http.status_code", int64(resp.StatusCode)),
	)

	// Validação: Verifica se a resposta da API foi bem-sucedida
	if resp.StatusCode != http.StatusOK {
		// Lê o corpo da resposta para depuração adicional
		// Isso ajuda a entender o motivo da falha (API key inválida, cidade não encontrada, etc.)
		body, errBody := io.ReadAll(resp.Body)
		if errBody != nil {
			log.Printf("Erro ao ler o corpo da resposta: %v", errBody)
		} else {
			log.Printf("Falha na consulta do clima: status %d, resposta: %s", resp.StatusCode, string(body))
		}
		err := fmt.Errorf("weather lookup failed")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}

	// Decodifica a resposta JSON da API WeatherAPI
	var weatherResp WeatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&weatherResp); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}
	
	// Adiciona a temperatura obtida ao span para facilitar análise
	span.SetAttributes(
		attribute.Float64("weather.temperature_celsius", weatherResp.Current.TempC),
	)
	// Marca o span como bem-sucedido
	span.SetStatus(codes.Ok, "Temperatura obtida com sucesso")

	return weatherResp.Current.TempC, nil
}