# Build-Stage
FROM golang:alpine as builder

# Setze das Arbeitsverzeichnis innerhalb des Containers
WORKDIR /app

# Kopiere go.mod und go.sum in das Arbeitsverzeichnis
COPY go.mod go.sum ./

# Lade die Abhängigkeiten herunter
RUN go mod download

# Kopiere den Quellcode in das Arbeitsverzeichnis
COPY . .

# Kompiliere das Programm
RUN go build -o main ./...

# Runtime-Stage
FROM alpine:latest

# Füge CA-Zertifikate hinzu, falls erforderlich
RUN apk --no-cache add ca-certificates

# Setze das Arbeitsverzeichnis innerhalb des Containers
WORKDIR /app

# Kopiere die Binary vom Build-Stage in den neuen Container
COPY --from=builder /app/main /app/main

# Exponiere den Port, auf dem der Webserver laufen wird
EXPOSE 8080

# Führe das kompilierte Programm aus
CMD ["./main"]
