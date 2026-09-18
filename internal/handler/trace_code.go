package handler

import (
	"cc-052/internal/model"
	"cc-052/internal/service"
	"cc-052/pkg/regioncode"
	"cc-052/pkg/response"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TraceCodeHandler struct {
	svc *service.TraceCodeService
}

func NewTraceCodeHandler(svc *service.TraceCodeService) *TraceCodeHandler {
	return &TraceCodeHandler{svc: svc}
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

	codes, err := h.svc.GenerateCodes(batchID, req.Count, req.PackageUnitCode)
	if err != nil {
		if errors.Is(err, service.ErrInvalidPackageUnit) {
			response.BadRequest(c, err.Error())
			return
		}
		response.Forbidden(c, err.Error())
		return
	}
	response.Created(c, gin.H{"codes": codes, "count": len(codes), "package_unit": req.PackageUnitCode})
}

func (h *TraceCodeHandler) Trace(c *gin.Context) {
	code := c.Param("code")

	// 扫码地区只接受扫码端显式上报的行政区划码（如客户端定位授权后传入）。
	// 绝不使用 X-Forwarded-For / ClientIP 等请求来源信息冒充扫码地区——
	// 转发头可被任意伪造，且 IP 归属地也不等于扫码发生地。
	var region *string
	if r := c.Query("region"); r != "" {
		if !regioncode.Valid(r) {
			response.BadRequest(c, "invalid region code, expected a 6-digit administrative division code")
			return
		}
		region = &r
	}

	trace, err := h.svc.Trace(code, region)
	if err != nil {
		response.NotFound(c, "trace code not found")
		return
	}
	response.Success(c, trace)
}

func (h *TraceCodeHandler) Validate(c *gin.Context) {
	code := c.Param("code")
	valid := h.svc.ValidateTraceCode(code)
	response.Success(c, gin.H{"valid": valid})
}
