package request

// WithLogging enables request/response logging.
// Optional redactHeaders specifies headers to redact in logs.
func (r *Request) WithLogging(redactHeaders ...string) *Request {
	r.logging = true
	r.redactHeaders = redactHeaders
	return r
}
