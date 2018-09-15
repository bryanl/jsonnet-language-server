package server

import (
	"io"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	"go.uber.org/zap"
)

func initTracing(service string, logger *zap.Logger) (opentracing.Tracer, io.Closer) {
	sender, err := jaeger.NewUDPTransport("0.0.0.0:6831", 0)
	if err != nil {
		logger.Fatal("cannot initialize UDP sender", zap.Error(err))
	}

	reporter := jaeger.NewRemoteReporter(
		sender,
		jaeger.ReporterOptions.BufferFlushInterval(1*time.Second),
	)

	return jaeger.NewTracer(
		service,
		jaeger.NewConstSampler(true),
		reporter,
	)
}
