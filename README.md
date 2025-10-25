# CEP Weather — Entrega

Este repositório contém um microserviço em Go que recebe um CEP (query param `cep`) e retorna a temperatura atual (C/F/K) consultando uma API de clima.

Conteúdo da entrega

- Código-fonte completo (neste repositório).
- Testes automatizados para `internal/weather` (httptest).
- Suporte a execução local via Docker / docker-compose.
- Instruções de deploy para Google Cloud Run.

Requisitos

- Go 1.20+ (ou versão compatível instalada localmente).
- Docker & docker-compose para execução local em container.
- gcloud CLI configurado (se for usar o deploy para Cloud Run).

Executando testes locais

1. Na raiz do projeto rode:

```bash
go test ./...
```

Os testes de `internal/weather` usam um servidor HTTP fake (httptest) e não fazem chamadas externas.

Executando a aplicação localmente (sem Docker)

Defina a variável de ambiente `WEATHER_API_KEY` e execute:

Windows PowerShell
```powershell
$env:WEATHER_API_KEY = 'sua_chave'
go run ./cmd
```

Linux / macOS
```bash
export WEATHER_API_KEY="sua_chave"
go run ./cmd
```

A aplicação expõe a porta 8080. A URL de teste local (quando roda local): `http://localhost:8080/?cep=01001000`.

Executando com Docker / docker-compose

O `docker-compose.yml` carrega variáveis sensíveis a partir de um arquivo local `.env` (nunca comitado).
Antes de subir com Docker Compose, crie um `.env` a partir do exemplo e insira a sua chave:

Windows PowerShell
```powershell
Copy-Item .env.example .env
notepad .env  # substituir REPLACE_WITH_YOUR_KEY pela sua chave real
docker-compose up --build
```

Linux / macOS
```bash
cp .env.example .env
# edite .env e substitua REPLACE_WITH_YOUR_KEY pela sua chave real
docker-compose up --build
```

O serviço ficará acessível em `http://localhost:8080`.

Deploy no Google Cloud Run (instruções)

1. Faça o build e envie a imagem para o Container Registry / Artifact Registry:

```powershell
gcloud builds submit --tag gcr.io/<PROJECT_ID>/cep-weather
```

2. Faça o deploy no Cloud Run (substitua `<PROJECT_ID>` e ajuste a região se necessário). Defina a variável de ambiente `WEATHER_API_KEY` durante o deploy:

```powershell
gcloud run deploy cep-weather --image gcr.io/<PROJECT_ID>/cep-weather --region us-central1 --platform managed --set-env-vars WEATHER_API_KEY=seu_valor_de_api_key
```

3. Verifique a variável de ambiente configurada e a URL do serviço:

```powershell
gcloud run services describe cep-weather --region us-central1 --format="yaml"
```

Logs

- Para ver logs históricos (ex.: últimos 1h) via gcloud:

```powershell
gcloud logging read 'resource.type="cloud_run_revision" AND resource.labels.service_name="cep-weather"' --limit=200 --freshness=1h --project=<PROJECT_ID> --format="yaml"
```

- Para ver apenas as mensagens do stdout (where the app prints diagnostics):

```powershell
gcloud logging read 'resource.type="cloud_run_revision" AND resource.labels.service_name="cep-weather" AND logName="projects/<PROJECT_ID>/logs/run.googleapis.com%2Fstdout"' --limit=200 --project=<PROJECT_ID>
```

Notas úteis

- O código espera por um query param `cep` com exatamente 8 dígitos: `?cep=01001000`.
- Se preferir que o serviço aceite CEPs formatados (ex.: `01001-000`) posso aplicar uma normalização no `cmd/main.go`.
- Os testes adicionados cobrem casos de sucesso, erro da API e falta de chave.
- Onde obter a API key: este projeto usa a WeatherAPI (ex.: https://www.weatherapi.com/). Registre-se lá para obter a sua chave de teste/produção.
- Segurança: NÃO comite o arquivo `.env` com chaves reais. Use `.env.example` com placeholders e inclua `.env` no `.gitignore` (já configurado).

Contato

Se quiser, eu posso também aplicar pequenas melhorias (fallback `API_KEY`, logs adicionais no handler) e preparar um script automatizado de CI para rodar os testes. Basta pedir.
