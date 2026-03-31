package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	pyroscope "github.com/grafana/pyroscope-go"
	"github.com/ikristina/go-worker-pool/metrics"
	"github.com/ikristina/go-worker-pool/task"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const NUM_JOBS = 5    // total number of jobs (buffer)
const NUM_WORKERS = 3 // concurrency limit

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("localhost:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("go-worker-pool"),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}


func worker(ctx context.Context, id int, jobs <-chan task.Job, results chan<- task.Result, wg *sync.WaitGroup, m *metrics.Metrics) {
	defer wg.Done() // the worker must signal that it finished

	for {
		select {
		case <-ctx.Done():
			// In case of context timeout
			return
		case j, ok := <-jobs:
			if !ok {
				return //channel is closed
			}
			slog.Info("worker started job", "worker", id, "job", j.ID)
			tracer := otel.Tracer("worker")
			jobCtx, span := tracer.Start(ctx, "process_job")
			span.SetAttributes(
				attribute.Int("worker.id", id),
				attribute.Int("job.id", j.ID),
				attribute.String("job.url", j.URL),
			)
			m.WorkersActive.Inc()
			start := time.Now()
			r := task.Run(jobCtx, &j)
			m.JobDuration.Observe(time.Since(start).Seconds())
			m.WorkersActive.Dec()
			if r.Err != nil {
				m.JobsTotal.WithLabelValues("failed").Inc()
				span.SetStatus(codes.Error, r.Err.Error())
			} else {
				m.JobsTotal.WithLabelValues("success").Inc()
				span.SetStatus(codes.Ok, "")
			}
			span.End()
			select {
			case results <- *r: // put the result on the channel
			case <-ctx.Done():
				return
			}
		}
	}
}

func main() {
	logFile, err := os.OpenFile("logs/app.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("failed to open log file", "err", err)
		os.Exit(1)
	}
	defer logFile.Close()

	logger := slog.New(slog.NewTextHandler(io.MultiWriter(os.Stderr, logFile), &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	profiler, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: "go-worker-pool",
		ServerAddress:   "http://localhost:4040",
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
		},
	})
	if err != nil {
		slog.Error("failed to start profiler", "err", err)
		os.Exit(1)
	}
	defer profiler.Stop()

	tp, err := initTracer(context.Background())
	if err != nil {
		slog.Error("failed to initialize tracer", "err", err)
		os.Exit(1)
	}
	defer tp.Shutdown(context.Background())

	// Create a non-global registry.
	reg := prometheus.NewRegistry()
	// Create new metrics and register them using the custom registry.
	m := metrics.NewMetrics(reg)

	// Expose metrics and custom registry via an HTTP server
	// using the HandleFor function. "/metrics" is the usual endpoint for that.
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server listen failed", "err", err)
			os.Exit(1)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := make(chan task.Job, NUM_JOBS)
	results := make(chan task.Result, NUM_JOBS)

	reg.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "jobs_queue_depth",
		Help: "Number of jobs currently waiting in the queue.",
	}, func() float64 { return float64(len(jobs)) }))

	var wg sync.WaitGroup
	// Start workers
	for w := 1; w <= NUM_WORKERS; w++ {
		wg.Add(1)
		go worker(ctx, w, jobs, results, &wg, m)
	}
	// Start producer goroutine

	go func() {
		urls := []string{
			"https://go.dev",
			"bobobo",
			"https://google.com",
			"https://github.com",
			"https://pkg.go.dev",
			"https://opensource.org",
		}
		for i, url := range urls {
			jobs <- task.Job{ID: i + 1, URL: url}
		}
		close(jobs)
	}()

	// Result collection (consumer goroutine)
	done := make(chan struct{}) // to signal completion (struct{} is 0 memory while bool or int would take at least 1 or 8 bytes)

	go func() {
		for res := range results {
			if res.Err != nil {
				slog.Error("job failed", "job", res.ID, "err", res.Err)
			} else {
				slog.Info("job succeeded", "job", res.ID, "value", res.Value)
			}
		}
		close(done)
	}()

	wg.Wait() // wait for workers to finish
	close(results)

	// wait for the collector to finish printing everything
	<-done
	slog.Info("all systems shut down")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	slog.Info("metrics available at http://localhost:8080/metrics — press Ctrl+C to exit")
	<-quit
	slog.Info("shutting down")
}
