FROM golang:1.23 as builder

WORKDIR /app

# Copia os arquivos go.mod e go.sum e baixa as dependências
COPY go.mod ./
RUN go mod download

# Copia o código fonte
COPY . .

# Compila a aplicação
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root
COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]
