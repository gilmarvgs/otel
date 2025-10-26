// Pacote telemetry fornece funcionalidades para configuração e inicialização do OpenTelemetry
package telemetry

// Importação dos pacotes necessários do OpenTelemetry
import (
	"go.opentelemetry.io/otel"                         // Pacote principal do OpenTelemetry
	"go.opentelemetry.io/otel/exporters/zipkin"        // Exportador para o Zipkin
	"go.opentelemetry.io/otel/sdk/resource"            // Recursos do SDK
	sdktrace "go.opentelemetry.io/otel/sdk/trace"      // SDK de rastreamento
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0" // Convenções semânticas
)

// InitTracer inicializa e configura o provedor de rastreamento do OpenTelemetry
// Recebe o nome do serviço como parâmetro e retorna um provedor de rastreamento configurado
func InitTracer(serviceName string) (*sdktrace.TracerProvider, error) {
	// Cria um exportador Zipkin que enviará os dados para a URL especificada
	exporter, err := zipkin.New(
		"http://localhost:9411/api/v2/spans", // URL do servidor Zipkin
	)
	if err != nil {
		return nil, err
	}

	// Cria um recurso com atributos que identificam o serviço
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,                          // URL do esquema de convenções semânticas
		semconv.ServiceNameKey.String(serviceName), // Define o nome do serviço
	)

	// Cria um provedor de rastreamento com as configurações necessárias
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),                // Configura o exportador
		sdktrace.WithResource(resource),               // Adiciona os recursos
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // Amostra todos os traces
	)

	// Define o provedor de rastreamento como global para toda a aplicação
	otel.SetTracerProvider(tp)

	return tp, nil
}
