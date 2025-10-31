// Pacote telemetry fornece funcionalidades para configuração e inicialização do OpenTelemetry
// Este pacote centraliza toda a configuração de rastreamento distribuído com Zipkin
package telemetry

// Importação dos pacotes necessários do OpenTelemetry
import (
	"os"
	
	"go.opentelemetry.io/otel"                         // Pacote principal do OpenTelemetry (tracer global)
	"go.opentelemetry.io/otel/exporters/zipkin"        // Exportador para enviar traces ao Zipkin
	"go.opentelemetry.io/otel/sdk/resource"            // Recursos do SDK (metadados do serviço)
	sdktrace "go.opentelemetry.io/otel/sdk/trace"      // SDK de rastreamento (TracerProvider)
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0" // Convenções semânticas (padrões de atributos)
)

// InitTracer inicializa e configura o provedor de rastreamento do OpenTelemetry
// 
// Esta função configura todo o sistema de rastreamento distribuído:
// - Conecta ao Zipkin para visualização de traces
// - Configura amostragem (sempre amostra todos os traces)
// - Define metadados do serviço para identificação
//
// Parâmetros:
//   - serviceName: Nome do serviço (ex: "service-a", "service-b")
//     Este nome aparecerá no Zipkin para identificar os traces
//
// Retorna:
//   - *sdktrace.TracerProvider: Provedor de rastreamento configurado
//   - error: Erro caso a configuração falhe
func InitTracer(serviceName string) (*sdktrace.TracerProvider, error) {
	// Obtém a URL do Zipkin das variáveis de ambiente
	// Permite configurar dinamicamente a URL do Zipkin (útil para diferentes ambientes)
	zipkinURL := os.Getenv("ZIPKIN_URL")
	if zipkinURL == "" {
		// URL padrão para ambiente Docker
		// No docker-compose, o serviço Zipkin está disponível em "zipkin:9411"
		zipkinURL = "http://zipkin:9411/api/v2/spans"
	}
	
	// Cria um exportador Zipkin que enviará os traces para a URL especificada
	// O exportador é responsável por serializar e enviar os spans ao Zipkin
	exporter, err := zipkin.New(zipkinURL)
	if err != nil {
		return nil, err
	}

	// Cria um recurso com atributos que identificam o serviço
	// Esses atributos serão adicionados a todos os spans gerados pelo serviço
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,                          // URL do esquema de convenções semânticas (padrão OTEL)
		semconv.ServiceNameKey.String(serviceName), // Define o nome do serviço para identificação no Zipkin
	)

	// Cria um provedor de rastreamento com as configurações necessárias
	// O TracerProvider é responsável por criar tracers e gerenciar o ciclo de vida dos spans
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),                // Configura o exportador (envia spans em lotes para eficiência)
		sdktrace.WithResource(resource),               // Adiciona os recursos (metadados do serviço)
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // Amostra todos os traces (100% das requisições são rastreadas)
	)

	// Define o provedor de rastreamento como global para toda a aplicação
	// Isso permite que qualquer parte do código use otel.Tracer() para criar spans
	otel.SetTracerProvider(tp)

	return tp, nil
}
