// Code generated from docs/openapi.yaml; DO NOT EDIT.
package api

type ResponseKind string

const (
	ResponseKindStandard     ResponseKind = "standard"
	ResponseKindError        ResponseKind = "error"
	ResponseKindVerifySource ResponseKind = "verify_source"
)

type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type VerifySource struct {
	Source   string `json:"source"`
	Verified bool   `json:"verified"`
	Reason   string `json:"reason,omitempty"`
}

type Meta struct {
	RequestID string `json:"request_id,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
	Count     int    `json:"count,omitempty"`
}

type Response struct {
	Kind         ResponseKind  `json:"kind"`
	Data         any           `json:"data,omitempty"`
	Error        *Error        `json:"error,omitempty"`
	VerifySource *VerifySource `json:"verify_source,omitempty"`
	Meta         *Meta         `json:"meta,omitempty"`
}
