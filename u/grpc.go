package u

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

// GRPCClientZapLogOption almost the same compare to grpc_zap.DefaultMessageProducer. Additionally, log traceID.
func GRPCClientZapLogOption() grpc_zap.Option {
	return grpc_zap.WithMessageProducer(func(ctx context.Context, msg string, level zapcore.Level, code codes.Code, err error, duration zapcore.Field) {
		traceID := TraceIDFromOutgoing(ctx)

		//Infof("[%s] msg=%s; code=%v; duration=%vms; err=%v", traceID, msg, code, float32(duration.Integer/1000)/1000, err)
		ctxzap.Extract(ctx).Check(level, msg).Write(
			zap.Error(err),
			zap.String("grpc.code", code.String()),
			duration,
			zap.String("trace_id", traceID),
		)
	})
}

// GRPCServerZapLogOption almost the same compare to grpc_zap.DefaultMessageProducer. Additionally, log traceID.
func GRPCServerZapLogOption() grpc_zap.Option {
	return grpc_zap.WithMessageProducer(func(ctx context.Context, msg string, level zapcore.Level, code codes.Code, err error, duration zapcore.Field) {
		traceID := TraceIDFromIncoming(ctx)

		ctxzap.Extract(ctx).Check(level, msg).Write(
			zap.Error(err),
			zap.String("grpc.code", code.String()),
			duration,
			zap.String("trace_id", traceID),
		)
	})
}

// AddHeaderToGRPCRequest is an alias to metadata.AppendToOutgoingContext in case you don't know how to add header to GRPC request.
// Now you know, just call metadata.AppendToOutgoingContext.
func AddHeaderToGRPCRequest(context context.Context, kv ...string) context.Context {
	return metadata.AppendToOutgoingContext(context, kv...)
}
