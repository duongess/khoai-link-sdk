package khoailinksdk

import (
	"github.com/khoai-link-protocol/core"
)

// Su dung truc tiep tu protocol de dam bao tinh nhat quan tren toan mang
type Response = core.Response

func NewSuccessResponse(reqID string, data any) *Response {
	return core.NewSuccessResponse(reqID, data)
}

func NewErrorResponse(reqID string, code int, msg string) *Response {
	return core.NewErrorResponse(reqID, code, msg)
}
