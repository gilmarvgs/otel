# Sistema de Consulta de Temperatura por CEP

Este projeto consiste em dois microserviços que trabalham em conjunto para fornecer informações de temperatura com base em um CEP fornecido. O sistema utiliza OpenTelemetry para rastreamento distribuído e Zipkin para visualização dos traces.

## Requisitos

- Docker e Docker Compose
- Chave de API do WeatherAPI (https://www.weatherapi.com/)

## Configuração

1. Clone o repositório:
```bash
git clone https://github.com/gilmarvgs/cep-weather.git
cd cep-weather
```

2. Configure a variável de ambiente da API do WeatherAPI:
```bash
export WEATHER_API_KEY=sua_chave_api_aqui
```

## Executando o Projeto

1. Inicie os serviços usando Docker Compose:
```bash
docker-compose up --build
```

Isso irá iniciar:
- Service A na porta 8080
- Service B na porta 8081
- Zipkin na porta 9411

## Como Usar

1. Para consultar a temperatura de um CEP, envie uma requisição POST para o Service A:

```bash
curl -X POST http://localhost:8080/weather \
  -H "Content-Type: application/json" \
  -d '{"cep": "29902555"}'
```

### Exemplo de Resposta de Sucesso:

```json
{
  "city": "São Paulo",
  "temp_C": 28.5,
  "temp_F": 83.3,
  "temp_K": 301.5
}
```

### Possíveis Códigos de Erro:

- 422: CEP inválido
- 404: CEP não encontrado
- 500: Erro interno do servidor

## Monitoramento e Tracing

O sistema utiliza OpenTelemetry para gerar traces distribuídos que podem ser visualizados no Zipkin:

1. Acesse o Zipkin UI: http://localhost:9411
2. Use a interface para visualizar os traces das requisições

## Estrutura do Projeto

- `service-a/`: Serviço responsável pelo input e validação do CEP
- `service-b/`: Serviço responsável pela consulta de localização e temperatura
- `internal/`: Pacotes compartilhados entre os serviços
  - `location/`: Cliente para a API ViaCEP
  - `weather/`: Cliente para a API WeatherAPI
  - `telemetry/`: Configuração do OpenTelemetry

## Desenvolvimento

Para executar o projeto em ambiente de desenvolvimento sem Docker:

1. Instale as dependências:
```bash
go mod download
```

2. Execute o Service B:
```bash
WEATHER_API_KEY=sua_chave_api PORT=8081 go run service-b/main.go
```

3. Em outro terminal, execute o Service A:
```bash
PORT=8080 SERVICE_B_URL=http://localhost:8081/weather go run service-a/main.go
```