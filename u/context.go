package u

import (
	"context"
	"sync"

	"google.golang.org/grpc/metadata"
)

// CTX is short for Context. Name is different from context.Context or gin.Context for preventing confusion.
type CTX struct {
	// traceID could be used for tracing in one service call life-time even across services.
	// Try to print log with a traceID for log-tracing convenience.
	traceID     string
	traceIDOnce sync.Once

	// Sometimes, developer just hard to make a decision to choose between panic and return-error.
	// In this case, try using PoR or PoRErr.
	// PreferPanic is a preference to PoR and PoRErr.
	PreferPanic bool

	// See Set and Get.
	kv map[string]interface{}
}

// TraceID returns TraceID. Create one if not.
func (c *CTX) TraceID() string {
	if c.traceID == "" {
		c.traceIDOnce.Do(func() {
			c.traceID = UUID12()
		})
	}
	return c.traceID
}

// PoR Panic or return the error referred by PreferPanic.
func (c *CTX) PoR(erro ErrorType) ErrorType {
	if c.PreferPanic {
		panic(erro)
	}
	return erro
}

// PoRErr Panic or return the error referred by PreferPanic.
func (c *CTX) PoRErr(err error) error {
	if c.PreferPanic {
		panic(err)
	}
	return err
}

// Get returns the value for the given key.
// If the value does not exist it returns nil.
func (c *CTX) Get(key string) interface{} {
	v, ok := c.kv[key]
	if !ok {
		return nil
	}
	return v
}

// Set is used to store a new key/value pair exclusively for this context.
func (c *CTX) Set(key string, value interface{}) {
	c.kv[key] = value
}

// CreateGRPCContext create a context.Context with header "trace_id".
func (c *CTX) CreateGRPCContext() context.Context {
	ctx := context.Background()
	return c.FillGRPCContext(ctx)
}

// FillGRPCContext append "trace_id" to context.Context .
func (c *CTX) FillGRPCContext(context context.Context) context.Context {
	return ContextByAppendingTraceID(context, c.traceID)
}

func ContextByAppendingTraceID(context context.Context, traceID string) context.Context {
	context = metadata.AppendToOutgoingContext(context, "trace_id", traceID)
	return context
}

//func CreateGRPCOutgoingContext(in context.Context) context.Context {
//	traceID := TraceIDFromIncoming(in)
//	out := metadata.AppendToOutgoingContext(context.Background(), "traceid", traceID)
//	return out
//}

// TraceIDFromOutgoing extract TraceID from context. return "" if not found.
func TraceIDFromOutgoing(context context.Context) string {
	md, ok := metadata.FromOutgoingContext(context)
	if ok {
		return TraceIDFromMD(md)
	}
	return ""
}

// TraceIDFromIncoming extract TraceID from context. return "" if not found.
func TraceIDFromIncoming(context context.Context) string {
	md, ok := metadata.FromIncomingContext(context)
	if ok {
		return TraceIDFromMD(md)
	}
	return ""
}

func TraceIDFromMD(md metadata.MD) string {
	traceID := md.Get("trace_id")
	if len(traceID) > 0 {
		return traceID[0]
	}
	return ""
}

func NewContext() *CTX {
	return &CTX{
		PreferPanic: true,
		kv:          map[string]interface{}{},
	}
}

func NewCTXWithTraceID(traceID string) *CTX {
	c := NewContext()
	c.traceID = traceID
	return c
}

func NewCTXWithGRPCContext(context context.Context) *CTX {
	traceID := TraceIDFromIncoming(context)
	c := NewContext()
	c.traceID = traceID
	return c
}
