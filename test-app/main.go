package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer trace.Tracer
	meter  metric.Meter
	logger *slog.Logger

	// In-memory document store
	documents   = make(map[string]Document)
	documentsMu sync.RWMutex
	docCounter  int

	// Metrics
	requestCounter  metric.Int64Counter
	activeDocGauge  metric.Int64UpDownCounter
	requestDuration metric.Float64Histogram
)

type Document struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateDocRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	TraceID string `json:"trace_id,omitempty"`
}

func initOtel(ctx context.Context) (func(), error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://alloy:4318"
	}
	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "test-api"
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.DeploymentEnvironmentKey.String("local"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Trace exporter
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(stripProtocol(endpoint)),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)

	// Metric exporter
	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(stripProtocol(endpoint)),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(10*time.Second))),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	// Log exporter
	logExporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(stripProtocol(endpoint)),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create log exporter: %w", err)
	}
	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)

	// Set up globals
	tracer = tp.Tracer(serviceName)
	meter = mp.Meter(serviceName)
	logger = slog.New(otelslog.NewHandler(serviceName, otelslog.WithLoggerProvider(lp)))

	// Cleanup function
	cleanup := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tp.Shutdown(shutdownCtx)
		_ = mp.Shutdown(shutdownCtx)
		_ = lp.Shutdown(shutdownCtx)
	}

	return cleanup, nil
}

func stripProtocol(endpoint string) string {
	for _, prefix := range []string{"http://", "https://"} {
		if len(endpoint) > len(prefix) && endpoint[:len(prefix)] == prefix {
			return endpoint[len(prefix):]
		}
	}
	return endpoint
}

func initMetrics() error {
	var err error
	requestCounter, err = meter.Int64Counter("http.requests.total",
		metric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		return err
	}

	activeDocGauge, err = meter.Int64UpDownCounter("documents.active",
		metric.WithDescription("Number of active documents"),
	)
	if err != nil {
		return err
	}

	requestDuration, err = meter.Float64Histogram("http.request.duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	return nil
}

// --- Handlers ---

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func listDocuments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, span := tracer.Start(ctx, "listDocuments")
	defer span.End()

	start := time.Now()
	defer func() {
		requestDuration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(
			attribute.String("method", "GET"),
			attribute.String("endpoint", "/api/documents"),
		))
	}()
	requestCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("method", "GET"),
		attribute.String("endpoint", "/api/documents"),
	))

	documentsMu.RLock()
	docs := make([]Document, 0, len(documents))
	for _, doc := range documents {
		docs = append(docs, doc)
	}
	documentsMu.RUnlock()

	span.SetAttributes(attribute.Int("documents.count", len(docs)))
	logger.InfoContext(ctx, "listed documents", "count", len(docs))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docs)
}

func getDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")
	_, span := tracer.Start(ctx, "getDocument", trace.WithAttributes(attribute.String("document.id", id)))
	defer span.End()

	start := time.Now()
	defer func() {
		requestDuration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(
			attribute.String("method", "GET"),
			attribute.String("endpoint", "/api/documents/{id}"),
		))
	}()
	requestCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("method", "GET"),
		attribute.String("endpoint", "/api/documents/{id}"),
	))

	documentsMu.RLock()
	doc, exists := documents[id]
	documentsMu.RUnlock()

	if !exists {
		span.SetStatus(codes.Error, "document not found")
		logger.WarnContext(ctx, "document not found", "id", id)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "document not found",
			TraceID: span.SpanContext().TraceID().String(),
		})
		return
	}

	// Simulate some processing time
	simulateWork(ctx, "fetchDocument")

	logger.InfoContext(ctx, "fetched document", "id", id, "title", doc.Title)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func createDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, span := tracer.Start(ctx, "createDocument")
	defer span.End()

	start := time.Now()
	defer func() {
		requestDuration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(
			attribute.String("method", "POST"),
			attribute.String("endpoint", "/api/documents"),
		))
	}()
	requestCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("method", "POST"),
		attribute.String("endpoint", "/api/documents"),
	))

	var req CreateDocRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.SetStatus(codes.Error, "invalid request body")
		span.RecordError(err)
		logger.ErrorContext(ctx, "failed to decode request", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "invalid request body",
			TraceID: span.SpanContext().TraceID().String(),
		})
		return
	}

	if req.Title == "" {
		span.SetStatus(codes.Error, "title is required")
		logger.WarnContext(ctx, "validation failed: title is empty")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "title is required",
			TraceID: span.SpanContext().TraceID().String(),
		})
		return
	}

	// Simulate processing
	simulateWork(ctx, "validateDocument")
	simulateWork(ctx, "saveDocument")

	documentsMu.Lock()
	docCounter++
	id := fmt.Sprintf("doc-%d", docCounter)
	now := time.Now()
	doc := Document{
		ID:        id,
		Title:     req.Title,
		Content:   req.Content,
		CreatedAt: now,
		UpdatedAt: now,
	}
	documents[id] = doc
	documentsMu.Unlock()

	activeDocGauge.Add(ctx, 1)
	span.SetAttributes(
		attribute.String("document.id", id),
		attribute.String("document.title", req.Title),
	)
	logger.InfoContext(ctx, "created document", "id", id, "title", req.Title)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(doc)
}

func deleteDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")
	_, span := tracer.Start(ctx, "deleteDocument", trace.WithAttributes(attribute.String("document.id", id)))
	defer span.End()

	start := time.Now()
	defer func() {
		requestDuration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(
			attribute.String("method", "DELETE"),
			attribute.String("endpoint", "/api/documents/{id}"),
		))
	}()
	requestCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("method", "DELETE"),
		attribute.String("endpoint", "/api/documents/{id}"),
	))

	documentsMu.Lock()
	_, exists := documents[id]
	if exists {
		delete(documents, id)
	}
	documentsMu.Unlock()

	if !exists {
		span.SetStatus(codes.Error, "document not found")
		logger.WarnContext(ctx, "document not found for deletion", "id", id)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "document not found",
			TraceID: span.SpanContext().TraceID().String(),
		})
		return
	}

	activeDocGauge.Add(ctx, -1)
	logger.InfoContext(ctx, "deleted document", "id", id)

	w.WriteHeader(http.StatusNoContent)
}

// simulateWork creates a child span with random latency to simulate real work
func simulateWork(ctx context.Context, operation string) {
	_, span := tracer.Start(ctx, operation)
	defer span.End()

	delay := time.Duration(50+rand.Intn(150)) * time.Millisecond
	time.Sleep(delay)

	span.SetAttributes(attribute.Float64("duration_ms", float64(delay.Milliseconds())))
}

func backgroundWorker(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "background worker shutting down")
			return
		case <-ticker.C:
			jobID := rand.Intn(10000)
			jobCtx, span := tracer.Start(ctx, "backgroundJob", trace.WithAttributes(attribute.Int("job.id", jobID)))

			logger.InfoContext(jobCtx, "starting background processing job", "job_id", jobID)
			simulateWork(jobCtx, "processData")

			status := rand.Intn(10)
			if status < 2 {
				err := fmt.Errorf("connection timeout to downstream service")
				span.SetStatus(codes.Error, err.Error())
				span.RecordError(err)
				logger.ErrorContext(jobCtx, "background job failed", "job_id", jobID, "error", err)
			} else if status < 4 {
				logger.WarnContext(jobCtx, "background job completed with warnings", "job_id", jobID, "retries", 1)
			} else {
				logger.InfoContext(jobCtx, "background job completed successfully", "job_id", jobID)
			}
			span.End()
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Init OpenTelemetry
	cleanup, err := initOtel(ctx)
	if err != nil {
		log.Fatalf("failed to init OpenTelemetry: %v", err)
	}
	defer cleanup()

	// Init metrics
	if err := initMetrics(); err != nil {
		log.Fatalf("failed to init metrics: %v", err)
	}

	// Start background worker to generate more logs and traces
	go backgroundWorker(ctx)

	// Routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /api/documents", listDocuments)
	mux.HandleFunc("GET /api/documents/{id}", getDocument)
	mux.HandleFunc("POST /api/documents", createDocument)
	mux.HandleFunc("DELETE /api/documents/{id}", deleteDocument)

	// Wrap with OTel HTTP middleware (auto-instruments all routes)
	handler := otelhttp.NewHandler(mux, "test-api")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
		defer shutdownCancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	log.Printf("🚀 Test API running on http://0.0.0.0:%s", port)
	log.Printf("📊 OTLP endpoint: %s", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	log.Println("Endpoints:")
	log.Println("  GET    /health")
	log.Println("  GET    /api/documents")
	log.Println("  GET    /api/documents/{id}")
	log.Println("  POST   /api/documents")
	log.Println("  DELETE /api/documents/{id}")

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
