package handler

import (
	"cc-052/internal/model"
	"cc-052/internal/service"
	"cc-052/pkg/region"
	"cc-052/pkg/response"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TraceCodeHandler struct {
	svc      *service.TraceCodeService
	resolver *region.Resolver
}

func NewTraceCodeHandler(svc *service.TraceCodeService, resolver *region.Resolver) *TraceCodeHandler {
	return &TraceCodeHandler{svc: svc, resolver: resolver}
}

func (h *TraceCodeHandler) Generate(c *gin.Context) {
	batchID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid batch id")
		return
	}

	var req model.GenerateCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	codes, err := h.svc.GenerateCodes(batchID, req.Count)
	if err != nil {
		response.Forbidden(c, err.Error())
		return
	}
	response.Created(c, gin.H{"codes": codes, "count": len(codes)})
}

func (h *TraceCodeHandler) Trace(c *gin.Context) {
	code := c.Param("code")

	// 扫码地区只能由服务端解析：取真实对端 IP（RemoteIP 忽略可伪造的
	// X-Forwarded-For 等请求头），再查服务端配置的网段→地区映射；
	// 解析不到就是未知，绝不采信请求自报的来源。
	var scanRegion *string
	if r, ok := h.resolver.Resolve(c.RemoteIP()); ok {
		scanRegion = &r
	}

	trace, err := h.svc.Trace(code, scanRegion)
	if err != nil {
		if errors.Is(err, service.ErrTraceCodeNotFound) {
			response.NotFound(c, "trace code not found")
			return
		}
		response.InternalError(c, "trace query failed")
		return
	}
	response.Success(c, trace)
}

func (h *TraceCodeHandler) Validate(c *gin.Context) {
	code := c.Param("code")
	valid := h.svc.ValidateTraceCode(code)
	response.Success(c, gin.H{"valid": valid})
}
