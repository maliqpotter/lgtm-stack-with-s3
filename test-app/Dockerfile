FROM public.ecr.aws/docker/library/golang:1.24.13-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o test-api ./main.go

FROM public.ecr.aws/docker/library/alpine:latest AS final

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/test-api .

EXPOSE 8080

CMD ["./test-api"]
