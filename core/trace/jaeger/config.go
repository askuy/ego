package jaeger

import (
	"github.com/gotomicro/ego/core/app"
	"os"
	"time"

	"github.com/gotomicro/ego/core/conf"
	"github.com/gotomicro/ego/core/defers"
	"github.com/gotomicro/ego/core/elog"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jconfig "github.com/uber/jaeger-client-go/config"
)

// Config ...
type Config struct {
	ServiceName      string
	Sampler          *jconfig.SamplerConfig
	Reporter         *jconfig.ReporterConfig
	Headers          *jaeger.HeadersConfig
	EnableRPCMetrics bool
	tags             []opentracing.Tag
	options          []jconfig.Option
	PanicOnError     bool
}

// StdConfig ...
func StdConfig(name string) *Config {
	return RawConfig("ego.trace.jaeger")
}

// RawConfig ...
func RawConfig(key string) *Config {
	var config = DefaultConfig()
	if err := conf.UnmarshalKey(key, config); err != nil {
		elog.Panic("unmarshal key", elog.Any("err", err))
	}
	return config
}

// DefaultConfig ...
func DefaultConfig() *Config {
	agentAddr := "127.0.0.1:6831"
	headerName := "x-trace-id"
	if addr := os.Getenv("JAEGER_AGENT_ADDR"); addr != "" {
		agentAddr = addr
	}
	return &Config{
		ServiceName: app.Name(),
		Sampler: &jconfig.SamplerConfig{
			Type:  "const",
			Param: 0.001,
		},
		Reporter: &jconfig.ReporterConfig{
			LogSpans:            false,
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  agentAddr,
		},
		EnableRPCMetrics: true,
		Headers: &jaeger.HeadersConfig{
			TraceBaggageHeaderPrefix: "ctx-",
			TraceContextHeaderName:   headerName,
		},
		tags: []opentracing.Tag{
			{Key: "hostname", Value: app.HostName()},
		},
		PanicOnError: true,
	}
}

// WithTag ...
func (config *Config) WithTag(tags ...opentracing.Tag) *Config {
	if config.tags == nil {
		config.tags = make([]opentracing.Tag, 0)
	}
	config.tags = append(config.tags, tags...)
	return config
}

// WithOption ...
func (config *Config) WithOption(options ...jconfig.Option) *Config {
	if config.options == nil {
		config.options = make([]jconfig.Option, 0)
	}
	config.options = append(config.options, options...)
	return config
}

// Build ...
func (config *Config) Build(options ...jconfig.Option) opentracing.Tracer {
	var configuration = jconfig.Configuration{
		ServiceName: config.ServiceName,
		Sampler:     config.Sampler,
		Reporter:    config.Reporter,
		RPCMetrics:  config.EnableRPCMetrics,
		Headers:     config.Headers,
		Tags:        config.tags,
	}
	tracer, closer, err := configuration.NewTracer(config.options...)
	if err != nil {
		if config.PanicOnError {
			elog.Panic("new jaeger", elog.FieldMod("jaeger"), elog.FieldErr(err))
		} else {
			elog.Error("new jaeger", elog.FieldMod("jaeger"), elog.FieldErr(err))
		}
	}
	defers.Register(closer.Close)
	return tracer
}
