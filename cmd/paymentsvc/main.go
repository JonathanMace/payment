package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/JonathanMace/payment"
	stdopentracing "github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	"golang.org/x/net/context"
	xtr "github.com/JonathanMace/tracing-framework-go/xtrace/client"
	bot "github.com/JonathanMace/tracing-framework-go/opentracing"
)

const (
	ServiceName = "payment"
)

func init() {
	fmt.Println("Connecting to xtrace-server:5563")
	xtr.Connect("xtrace-server:5563")
	xtr.SetProcessName("Payment Microservice")
}

func main() {
	var (
		port          = flag.String("port", "8080", "Port to bind HTTP listener")
		zip           = flag.String("zipkin", os.Getenv("ZIPKIN"), "Zipkin address")
		declineAmount = flag.Float64("decline", 100, "Decline payments over certain amount")
	)
	flag.Parse()
	var tracer stdopentracing.Tracer
	{
		// Log domain.
		var logger log.Logger
		{
			logger = log.NewLogfmtLogger(xtr.MakeWriter(os.Stderr))
			logger = log.NewContext(logger).With("ts", log.DefaultTimestampUTC)
			logger = log.NewContext(logger).With("caller", log.DefaultCaller)
		}
		if *zip == "" {
			tracer = stdopentracing.NoopTracer{}
		} else {
			logger := log.NewContext(logger).With("tracer", "Zipkin")
			logger.Log("addr", zip)
			collector, err := zipkin.NewHTTPCollector(
				*zip,
				zipkin.HTTPLogger(logger),
			)
			if err != nil {
				logger.Log("err", err)
				os.Exit(1)
			}
			tracer, err = zipkin.NewTracer(
				zipkin.NewRecorder(collector, false, fmt.Sprintf("localhost:%v", port), ServiceName),
			)
			tracer = bot.Wrap(tracer.(zipkin.Tracer))
			if err != nil {
				logger.Log("err", err)
				os.Exit(1)
			}
		}
		stdopentracing.InitGlobalTracer(tracer)

	}
	// Mechanical stuff.
	errc := make(chan error)
	ctx := context.Background()

	handler, logger := payment.WireUp(ctx, float32(*declineAmount), tracer, ServiceName)

	// Create and launch the HTTP server.
	go func() {
		logger.Log("transport", "HTTP", "port", *port)
		errc <- http.ListenAndServe(":"+*port, handler)
	}()

	// Capture interrupts.
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	logger.Log("exit", <-errc)
}
