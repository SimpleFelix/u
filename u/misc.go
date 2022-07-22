package u

import (
	"encoding/json"
	"reflect"
	"strings"
	"unsafe"

	"github.com/google/uuid"
)

// AvroNameFor if given name does not start with [A-Za-z_], returned avro name will start with a prefix '_'.
// If given name is empty, returns "_".
func AvroNameFor(name string, allowedSubsequentRunes ...rune) (avro string) {
	if name == "" {
		return "_"
	}
	var sb strings.Builder
	first := name[0]
	if !isValidFirstAvroChar(first) {
		// 第一个字符不合法，则增加_前缀。
		sb.WriteString("_")
	}

	for _, r := range name {
		if isValidSubsequentAvroRune(r) {
			sb.WriteRune(r)
			continue
		}
		if isValidSubsequentRune(r, allowedSubsequentRunes...) {
			sb.WriteRune(r)
			continue
		}
		sb.WriteRune('_')
	}
	return sb.String()
}

func isValidSubsequentRune(r rune, allowedSubsequentRunes ...rune) bool {
	for _, a := range allowedSubsequentRunes {
		if r == a {
			return true
		}
	}
	return false
}

// 第2个字符到最后1个字符范围[a-zA-Z0-9_]
func isValidSubsequentAvroRune(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		r == '_' ||
		(r >= '0' && r <= '9')
}

// 第1个字符范围[a-zA-Z_]
func isValidFirstAvroChar(r uint8) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		r == '_'
}

func IsAvroName(name string, allowedSubsequentRunes ...rune) bool {
	if name == "" {
		return false
	}

	first := name[0]
	if !isValidFirstAvroChar(first) {
		return false
	}

	for _, r := range name {
		if isValidSubsequentAvroRune(r) {
			continue
		}
		if isValidSubsequentRune(r, allowedSubsequentRunes...) {
			continue
		}
		return false
	}
	return true
}

// GetFieldValueByName is a draft, need test
func GetFieldValueByName(obj interface{}, name string) interface{} {
	contextValues := reflect.ValueOf(obj).Elem()
	contextKeys := reflect.TypeOf(obj).Elem()

	for i := 0; i < contextValues.NumField(); i++ {
		reflectField := contextKeys.Field(i)
		if reflectField.Name != name {
			continue
		}
		reflectValue := contextValues.Field(i)
		reflectValue = reflect.NewAt(reflectValue.Type(), unsafe.Pointer(reflectValue.UnsafeAddr())).Elem()
		return reflectValue.Interface()
	}
	return nil
}

// UUID12 returns a length of 12 characters UUID string.
func UUID12() string {
	return uuid.NewString()[24:]
}

// UUID8 returns a length of 8 characters UUID string.
func UUID8() string {
	return uuid.NewString()[:8]
}

// UUID4 returns a length of 4 characters UUID string.
func UUID4() string {
	return uuid.NewString()[:4]
}

// ShortUUID returns a certain length of UUID string.
// length must between [1, 32].
// There is no '-' in returned uuid.
// Will panic if length is not in the range.
func ShortUUID(length int) string {
	if length > 32 || length <= 0 {
		panic(ErrShortUUIDLenConstraint())
	}
	u := uuid.NewString()
	if length <= 8 {
		return u[:length]
	}
	if length <= 12 {
		return u[24 : 24+length]
	}
	u = strings.ReplaceAll(u, "-", "")
	return u[:length]
}

func AutoRecover(ctx *CTX, job func()) {
	defer func() {
		if err := recover(); err != nil {
			var traceID string
			if ctx != nil {
				traceID = ctx.traceID
			}
			Errorf("[%v] %v", traceID, err)
		}
	}()
	job()
}

func AutoRecoverReturns[T any](ctx *CTX, job func() T) T {
	defer func() {
		if err := recover(); err != nil {
			var traceID string
			if ctx != nil {
				traceID = ctx.traceID
			}
			Errorf("[%v] %v", traceID, err)
		}
	}()
	return job()
}

func AutoRecoverAsync(ctx *CTX, job func()) {
	go func() {
		AutoRecover(ctx, job)
	}()
}

func IsValueNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr,
		reflect.Interface,
		reflect.Map,
		reflect.Slice,
		reflect.Func,
		reflect.Chan,
		reflect.UnsafePointer:
		return reflect.ValueOf(i).IsNil()
	}
	return false
}

func MapToType(m map[string]interface{}, t interface{}) {
	bytes, err := json.Marshal(m)
	if err != nil {
		panic(ErrFailedToMarshalJSON(err))
	}

	err = json.Unmarshal(bytes, &t)
	if err != nil {
		panic(ErrFailedToUnmarshalJSON(err))
	}
}
