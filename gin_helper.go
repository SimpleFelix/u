package u

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"unsafe"

	"github.com/gin-gonic/gin"
)

const TraceIDKey = "TID"

func traceIDForGinCreateIfNil(c *gin.Context) (traceID string) {
	if c == nil {
		return "gin_Context_must_not_be_nil!"
	}
	value, ok := c.Get(TraceIDKey)
	needCreate := false
	if ok {
		traceID, ok = value.(string)
		if !ok {
			needCreate = true
		}
	} else {
		needCreate = true
	}

	if needCreate {
		traceID = UUID12()
		c.Set(TraceIDKey, traceID)
	}
	return traceID
}

func traceIDFromGin(c *gin.Context) (traceID string) {
	value, ok := c.Get(TraceIDKey)
	if ok {
		traceID, _ = value.(string)
	}
	return traceID
}

// GinHelper provides some helper functions. Respond JSON only.
type GinHelper struct {
	*gin.Context
	//ctx *CTX
}

//var g = GinHelper{}
//
//func G() GinHelper {
//	return g
//}

func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		injectCTX(c)
		defer handlePanic(c)
		c.Next()
	}
}

func injectCTX(c *gin.Context) *CTX {
	if _, ok := c.Get("ctx"); ok {
		return nil
	}

	ctx := &CTX{
		traceID: UUID12(),
	}
	c.Set("ctx", ctx)

	return ctx
}

func getCTX(c *gin.Context) *CTX {
	if v, ok := c.Get("ctx"); ok {
		if ctx, ok := v.(*CTX); ok {
			return ctx
		}
	}

	return injectCTX(c)
}

func NewGinHelper(c *gin.Context) *GinHelper {
	//traceID := traceIDForGinCreateIfNil(c)
	return &GinHelper{
		Context: c,
		//ctx:     NewCTXWithTraceID(traceID),
	}
}

func (r *GinHelper) CTX() *CTX {
	return getCTX(r.Context)
}

type ErrorPayload struct {
	Code interface{} `json:"code,omitempty"`
	Desc string      `json:"desc"`
	TID  string      `json:"tid,omitempty"`
}

type KV = map[string]interface{}

//type KV struct {
//	k string
//	v interface{}
//}
//
//func NewKV(key string, value interface{}) KV {
//	return KV{k: key, v: value}
//}

const errorKey = "error"

func commonResponseBody() map[string]interface{} {
	return map[string]interface{}{
		errorKey: nil,
	}
}

// MustBind binds parameters to obj which must be a pointer. If any error occurred, respond 400.
// return true if binding succeed, vice versa.
func (r *GinHelper) MustBind(obj interface{}) bool {
	if err := r.BindUri(obj); err != nil {
		r.RespondError(ErrParamBindingErr(err))
		return false
	}
	if err := r.ShouldBind(obj); err != nil {
		r.RespondError(ErrParamBindingErr(err))
		return false
	}
	return true
}

// Bind Deprecated. binds parameters to obj which must be a pointer. If any error occurred, respond 400.
// return true if binding succeed, vice versa.
func (r *GinHelper) Bind(obj interface{}) bool {
	_ = r.ShouldBindUri(obj)
	if err := r.ShouldBind(obj); err != nil {
		r.RespondError(ErrParamBindingErr(err))
		return false
	}
	return true
}

func respondJSON(c *gin.Context, status int, body interface{}) {
	if c == nil {
		Errorf("calling respondJSON(*gin.Context, status, body) with nil context")
		return
	}
	c.JSON(status, body)
}

// Respond Example: payload 1 is {k: "msg" v: "ok"}; payload 2 is {k: "data" v:{id: 1}}.
// Response JSON will be
//
//	{
//		"error": null,
//		"msg": "ok",
//		"data": {
//			"id": 1
//		}
//	}
func (r *GinHelper) Respond(status int, payload KV) {
	body := commonResponseBody()
	for k, v := range payload {
		body[k] = v
	}
	respondJSON(r.Context, status, body)
}

// respondError respond custom status code with error in the response JSON
func (r *GinHelper) RespondError(erro ErrorType) {
	respondError(r.Context, erro)
}

// respondError respond custom status code with error in the response JSON
func respondError(gc *gin.Context, erro ErrorType) {
	if gc == nil {
		Errorf("calling respondError(*gin.Context, erro) with nil context")
		return
	}
	if erro == nil {
		Errorf("calling respondError(*gin.Context, erro) with nil error")
		gc.Abort()
	}

	// Discussion
	// Code in this function may panic.
	// For example, if erro is (*ErrorType, nil), call erro's function can either result success or failure, depends on each function's implementation.
	// If panicked, respondError will recover once, create an internalError object and call respondError with the new erro object.
	// If panicked again, respondError will give up. let http.serve() recover.
	defer func() {
		_, ok := gc.Get("respondErrorFailure")
		if ok {
			// We are here only because recovery below panicked.
			// If code panicked again. Process won't crash because http.serve() will recover.
			Errorf("[%s] respondError panicked twice. erro={type=%T; value=%v}", traceIDFromGin(gc), erro, erro)
			gc.Abort()
			return
		}

		if err := recover(); err != nil {
			// err could be (*ErrorType, nil)
			// So create a solid ErrorType object
			gc.Set("respondErrorFailure", true)
			e := ErrInternalError(fmt.Sprintf("Unparseable error. type=%T; value=%v", erro, erro))
			respondError(gc, e)
		}
	}()

	body := commonResponseBody()
	payload := ErrorPayload{
		Code: erro.ErrorCode(),
		Desc: erro.Error(),
	}

	payload.TID = traceIDForGinCreateIfNil(gc)
	body[errorKey] = payload

	if gin.IsDebugging() || (erro.Extra() != &notWorthLogging && erro.StatusCode() >= 500) {
		// get raw string of http request using reflect.
		requestLog := requestAsText(gc.Request)

		log := fmt.Sprintf("tid=%v; %scode=%v; error=%v; status=%v", payload.TID, requestLog, erro.ErrorCode(), erro.Error(), erro.StatusCode())

		if erro.Extra() == &printErrAsInfo {
			Info(log)
		} else {
			Error(log)
		}
	}

	respondJSON(gc, erro.StatusCode(), body)
}

var MaxLengthOfRequestDump = 4 * 1024

func requestAsText(request *http.Request) (requestLog string) {
	if request.ContentLength > 0 && request.Body != nil {
		// Possible to fail with future go version.
		v := reflect.ValueOf(request.Body).Elem()
		v = v.FieldByName("src").Elem().Elem()
		v = v.FieldByName("R").Elem().Elem()
		lv := v.FieldByName("w") // get length of package
		lv = reflect.NewAt(lv.Type(), unsafe.Pointer(lv.UnsafeAddr())).Elem()
		length := lv.Interface().(int)
		v = v.FieldByName("buf")
		v = reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
		buf, ok := v.Interface().([]byte)

		if length > MaxLengthOfRequestDump {
			length = MaxLengthOfRequestDump
		}

		if ok {
			requestLog = fmt.Sprintf("request=\n%s\n---EOR---\n", string(buf[:length]))
		}
	} else {
		// assemble headers
		headers := make([]string, 0, len(request.Header))
		for k, vs := range request.Header {
			for _, v := range vs {
				headers = append(headers, k+": "+v)
			}
		}
		requestLog = fmt.Sprintf(`request=
%s %s
%v
---EOR---
`, request.Method, request.RequestURI, strings.Join(headers, "\n"))
	}

	return
}

// RespondKV caller provides one object/array/slice and error, nil if no error.
// Function will make response properly.
func (r *GinHelper) RespondKV(successStatusCode int, key string, value interface{}, erro ErrorType) {
	if erro != nil {
		r.RespondError(erro)
		return
	}
	r.Respond(successStatusCode, KV{key: value})
}

func (r *GinHelper) RespondKV200(key string, value interface{}, erro ErrorType) {
	r.RespondKV(200, key, value, erro)
}

// RespondKVs caller provides multiple KV objects and error, nil if no error.
// Function will make response properly.
func (r *GinHelper) RespondKVs(successStatusCode int, erro ErrorType, payload KV) {
	if erro != nil {
		r.RespondError(erro)
		return
	}
	r.Respond(successStatusCode, payload)
}

func (r *GinHelper) RespondKVs200(erro ErrorType, payload KV) {
	r.RespondKVs(200, erro, payload)
}

// RespondFirst caller provide an slice/array. Only the first element if exists will be in the response JSON.
// todo generic
func (r *GinHelper) RespondFirst(successStatusCode int, key string, values interface{}, erro ErrorType) {
	if erro != nil {
		r.RespondError(erro)
		return
	}
	//todo 等范型推出后，删除反射代码。
	if reflect.TypeOf(values).Kind() != reflect.Slice {
		panic(fmt.Sprintf("GeneralRespondFirst values %v", values))
	}
	s := reflect.ValueOf(values)
	if s.Len() > 0 {
		r.Respond(successStatusCode, KV{key: s.Index(0).Interface()})
	} else {
		r.Respond(successStatusCode, KV{key: nil})
	}
}

func (r *GinHelper) RespondFirst200(key string, values interface{}, erro ErrorType) {
	r.RespondFirst(200, key, values, erro)
}

// RespondErrorElse if error is not nil, respond error.StatusCode() and error in the response JSON;
// Otherwise, respond successStatusCode and error: null in the response JSON
func (r *GinHelper) RespondErrorElse(successStatusCode int, erro ErrorType) {
	if erro != nil {
		r.RespondError(erro)
		return
	}
	r.Respond(successStatusCode, nil)
}

func (r *GinHelper) RespondErrorElse200(erro ErrorType) {
	r.RespondErrorElse(200, erro)
}

func (r *GinHelper) UnmarshalJSONToMap() (m map[string]interface{}, erro ErrorType) {
	bytes, err := io.ReadAll(r.Request.Body)
	if err != nil {
		return nil, ErrFailedToReadRequestBody(err)
	}

	if len(bytes) == 0 {
		m = map[string]interface{}{}
		return
	}

	err = json.Unmarshal(bytes, &m)
	if err != nil {
		return nil, ErrFailedToUnmarshalJSON(err)
	}

	return
}

func (r *GinHelper) BodyAsJSONSlice() (s []map[string]interface{}, erro ErrorType) {
	bytes, err := io.ReadAll(r.Request.Body)
	if err != nil {
		return nil, ErrFailedToReadRequestBody(err)
	}
	err = json.Unmarshal(bytes, &s)
	if err != nil {
		return nil, ErrFailedToUnmarshalJSON(err)
	}
	return s, nil
}

//func G(handle func(h *GinHelper)) gin.HandlerFunc {
//	return func(context *gin.Context) {
//		defer handlePanic(context)
//		h := NewGinHelper(context)
//		handle(h)
//	}
//}

func handlePanic(c *gin.Context) {
	if err := recover(); err != nil {
		erro, ok := err.(ErrorType)
		if ok {
			respondError(c, erro)
			c.Abort()
		} else {
			respondError(c, ErrAnyError(err))
			c.Abort()
			//Errorf("Gin catched a panic. traceID=%s; error=%v", traceIDFromGin(c), err)
			//c.AbortWithStatus(http.StatusInternalServerError)
		}
	}
}

// CreateGRPCContext create a context.Context with header "tid".
func (r *GinHelper) CreateGRPCContext() context.Context {
	context := context.Background()
	return getCTX(r.Context).FillGRPCContext(context)
}

func GetJWTClaims(c *gin.Context, claimsPointer any) {
	a := c.GetHeader("Authorization")
	//if !strings.HasPrefix(strings.ToLower(a), "Bearer ") {
	if !strings.HasPrefix(a, "Bearer ") {
		panic(ErrInvalidJWT("Invalid JWT."))
	}

	tokenString := a[7:]
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		panic(ErrInvalidJWT("Invalid JWT."))
	}

	claimBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		panic(ErrInvalidJWT(err))
	}

	err = json.Unmarshal(claimBytes, claimsPointer)
	if err != nil {
		panic(ErrInvalidJWT(err))
	}
}
