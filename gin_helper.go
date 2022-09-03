package u

import (
	"context"
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
		return "gin.Context_must_not_be_nil!"
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
	ctx *CTX
}

func NewGinHelper(c *gin.Context) GinHelper {
	traceID := traceIDForGinCreateIfNil(c)
	return GinHelper{
		Context: c,
		ctx:     NewCTXWithTraceID(traceID),
	}
}

func (h GinHelper) Ctx() *CTX {
	return h.ctx
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
func (h GinHelper) MustBind(obj interface{}) bool {
	if err := h.BindUri(obj); err != nil {
		h.RespondError(ErrParamBindingErr(err))
		return false
	}
	if err := h.ShouldBind(obj); err != nil {
		h.RespondError(ErrParamBindingErr(err))
		return false
	}
	return true
}

// Bind Deprecated. binds parameters to obj which must be a pointer. If any error occurred, respond 400.
// return true if binding succeed, vice versa.
func (h GinHelper) Bind(obj interface{}) bool {
	_ = h.ShouldBindUri(obj)
	if err := h.ShouldBind(obj); err != nil {
		h.RespondError(ErrParamBindingErr(err))
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

// Respond Example: playload1 is {k: "msg" v: "ok"} and payload2 is {k: "data" v:{id: 1}}.
// Response JSON will be
//
//	{
//		"error": null,
//		"msg": "ok",
//		"data": {
//			"id": 1
//		}
//	}
func (h GinHelper) Respond(status int, payload KV) {
	body := commonResponseBody()
	for k, v := range payload {
		body[k] = v
	}
	respondJSON(h.Context, status, body)
}

// respondError respond custom status code with error in the response JSON
func (h GinHelper) RespondError(erro ErrorType) {
	respondError(h.Context, erro)
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
		if reflect.TypeOf(erro).Name() == "u_internalError" {
			// We are here only because first recovery panicked.
			// Even below lines still panic. http.serve() will recover. Thus, service won't crash.
			Errorf("[%s] respondError panicked twice. erro={type=%T; value=%v}", traceIDFromGin(gc), erro, erro)
			gc.Abort()
			return
		}

		if err := recover(); err != nil {
			// err could be (*ErrorType, nil)
			// So create a solid ErrorType object
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

	if erro.Extra() != NotWorthLogging {
		// get raw string of http request using reflect.
		requestLog := requestAsText(gc.Request)

		log := fmt.Sprintf("tid=%v; %scode=%v; error=%v; status=%v", payload.TID, requestLog, erro.ErrorCode(), erro.Error(), erro.StatusCode())

		if erro.StatusCode() >= 500 && erro.Extra() != PrintErrAsInfo {
			Error(log)
		} else {
			Info(log)
		}
	}

	respondJSON(gc, erro.StatusCode(), body)
}

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
---EOR--
`, request.Method, request.RequestURI, strings.Join(headers, "\n"))
	}

	return
}

// RespondKV caller provides one object/array/slice and error, nil if no error.
// Function will make response properly.
func (h GinHelper) RespondKV(successStatusCode int, key string, value interface{}, erro ErrorType) {
	if erro != nil {
		h.RespondError(erro)
		return
	}
	h.Respond(successStatusCode, KV{key: value})
}

func (h GinHelper) RespondKV200(key string, value interface{}, erro ErrorType) {
	h.RespondKV(200, key, value, erro)
}

// RespondKVs caller provides multiple KV objects and error, nil if no error.
// Function will make response properly.
func (h GinHelper) RespondKVs(successStatusCode int, erro ErrorType, payload KV) {
	if erro != nil {
		h.RespondError(erro)
		return
	}
	h.Respond(successStatusCode, payload)
}

func (h GinHelper) RespondKVs200(erro ErrorType, payload KV) {
	h.RespondKVs(200, erro, payload)
}

// RespondFirst caller provide an slice/array. Only the first element if exists will be in the response JSON.
// todo generic
func (h GinHelper) RespondFirst(successStatusCode int, key string, values interface{}, erro ErrorType) {
	if erro != nil {
		h.RespondError(erro)
		return
	}
	//todo 等范型推出后，删除反射代码。
	if reflect.TypeOf(values).Kind() != reflect.Slice {
		panic(fmt.Sprintf("GeneralRespondFirst values %v", values))
	}
	s := reflect.ValueOf(values)
	if s.Len() > 0 {
		h.Respond(successStatusCode, KV{key: s.Index(0).Interface()})
	} else {
		h.Respond(successStatusCode, KV{key: nil})
	}
}

func (h GinHelper) RespondFirst200(key string, values interface{}, erro ErrorType) {
	h.RespondFirst(200, key, values, erro)
}

// RespondErrorElse if error is not nil, respond error.StatusCode() and error in the response JSON;
// Otherwise, respond successStatusCode and error: null in the response JSON
func (h GinHelper) RespondErrorElse(successStatusCode int, erro ErrorType) {
	if erro != nil {
		h.RespondError(erro)
		return
	}
	h.Respond(successStatusCode, nil)
}

func (h GinHelper) RespondErrorElse200(erro ErrorType) {
	h.RespondErrorElse(200, erro)
}

func (h GinHelper) BodyAsJSONMap() (m map[string]interface{}, erro ErrorType) {
	bytes, err := io.ReadAll(h.Request.Body)
	if err != nil {
		return nil, ErrFailedToReadRequestBody(err)
	}
	err = json.Unmarshal(bytes, &m)
	if err != nil {
		return nil, ErrFailedToUnmarshalJSON(err)
	}
	return m, nil
}

func (h GinHelper) BodyAsJSONSlice() (s []map[string]interface{}, erro ErrorType) {
	bytes, err := io.ReadAll(h.Request.Body)
	if err != nil {
		return nil, ErrFailedToReadRequestBody(err)
	}
	err = json.Unmarshal(bytes, &s)
	if err != nil {
		return nil, ErrFailedToUnmarshalJSON(err)
	}
	return s, nil
}

func G(handle func(h GinHelper)) gin.HandlerFunc {
	return func(context *gin.Context) {
		defer handlePanic(context)
		h := NewGinHelper(context)
		handle(h)
	}
}

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
func (h GinHelper) CreateGRPCContext() context.Context {
	context := context.Background()
	return h.Ctx().FillGRPCContext(context)
}
