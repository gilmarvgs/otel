# Estágio de compilação - Usa a imagem oficial do Go com Alpine Linux
FROM golang:1.21-alpine AS builder

# Define o diretório de trabalho dentro do container
WORKDIR /app

# Copia os arquivos de dependências e faz o download delas
COPY go.mod go.sum ./
RUN go mod download

# Copia todo o código fonte para o container
COPY . .

# Compila os serviços sem CGO e especificamente para Linux
RUN CGO_ENABLED=0 GOOS=linux go build -o service-a ./service-a
RUN CGO_ENABLED=0 GOOS=linux go build -o service-b ./service-b

# Estágio final - Usa uma imagem Alpine limpa
FROM alpine:latest

# Define o diretório de trabalho
WORKDIR /app

# Copia os binários compilados do estágio anterior
COPY --from=builder /app/service-a .
COPY --from=builder /app/service-b .

# Instala os certificados CA necessários para HTTPS
RUN apk --no-cache add ca-certificates

# Expõe as portas que os serviços usarão
EXPOSE 8080 8081
