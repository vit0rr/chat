package constants

type ctxKeyRequestID int
type ctxKeyPathPattern int
type ctxKeyStatusCode int
type ctxKeyMethod int
type ctxKeyRealIP int

const (
	// RequestID - Context key that holds the unique request ID in a request context.
	CtxKeyRequestID ctxKeyRequestID = 0
	// PathPattern - Context key that holds the path pattern in a request context. i.e /users/{id}
	CtxKeyPathPattern ctxKeyPathPattern = 0
	// StatusCode - Context key that holds the status code in a request context.
	CtxKeyStatusCode ctxKeyStatusCode = 0
	// Method - Context key that holds the method in a request context.
	CtxKeyMethod ctxKeyMethod = 0
	// RealIP - Context key that holds the real IP in a request context.
	CtxKeyRealIP ctxKeyRealIP = 0
)

// KeyName maps for logging purposes
var ContextKeys = []struct {
	Key   any
	Label string
}{
	{CtxKeyRequestID, "http.request_id"},
	{CtxKeyPathPattern, "http.path_pattern"},
	{CtxKeyStatusCode, "http.status_code"},
	{CtxKeyMethod, "http.method"},
	{CtxKeyRealIP, "http.real_ip"},
}
