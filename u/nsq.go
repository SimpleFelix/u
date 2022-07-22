package u

import (
	"github.com/nsqio/go-nsq"
	"go.uber.org/zap/zapcore"
)

func ZapLevelToNSQ(zl zapcore.Level) nsq.LogLevel {
	switch zl {
	case zapcore.DebugLevel:
		return nsq.LogLevelDebug
	case zapcore.InfoLevel:
		return nsq.LogLevelInfo
	case zapcore.WarnLevel:
		return nsq.LogLevelWarning
	case zapcore.ErrorLevel:
		return nsq.LogLevelError
	default:
		return nsq.LogLevelInfo
	}
}

func MustCreateNSQConsumer(topic string, handler nsq.Handler, channel, lookupdSvr string, logLevel nsq.LogLevel) *nsq.Consumer {
	cfg := nsq.NewConfig()
	c, err := nsq.NewConsumer(topic, channel, cfg)

	if err != nil {
		panic(err)
	}

	c.SetLoggerLevel(logLevel)

	c.AddHandler(handler)

	// Use nsqlookupd to discover nsqd instances.
	// See also ConnectToNSQD, ConnectToNSQDs, ConnectToNSQLookupds.
	err = c.ConnectToNSQLookupd(lookupdSvr)

	if err != nil {
		//todo retry?
		panic(err)
	}

	return c
}

func MustCreateNSQProducer(nsqdSvr string, logLevel nsq.LogLevel) *nsq.Producer {
	cfg := nsq.NewConfig()
	p, err := nsq.NewProducer(nsqdSvr, cfg)
	
	if err != nil {
		panic(err)
	}

	p.SetLoggerLevel(logLevel)

	return p
}
