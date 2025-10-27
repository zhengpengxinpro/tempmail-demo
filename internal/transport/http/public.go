package httptransport

import (
	"github.com/gin-gonic/gin"

	"tempmail/backend/internal/service"
)

// PublicHandler 公开API处理器（无需认证）
type PublicHandler struct {
	systemDomainService *service.SystemDomainService
}

// NewPublicHandler 创建公开API处理器
func NewPublicHandler(systemDomainService *service.SystemDomainService) *PublicHandler {
	return &PublicHandler{
		systemDomainService: systemDomainService,
	}
}

// GetAvailableDomains godoc
// @Summary 获取可用域名列表
// @Description 获取所有可用的系统域名列表（公开接口，无需认证）
// @Tags Public
// @Produce json
// @Success 200 {object} Response{data=object{domains=[]string,count=int}}
// @Router /v1/public/domains [get]
func (h *PublicHandler) GetAvailableDomains(c *gin.Context) {
	// 获取已激活的系统域名
	domains, err := h.systemDomainService.GetActiveDomains()
	if err != nil {
		InternalError(c, "获取可用域名失败")
		return
	}

	// 提取域名列表
	domainList := make([]string, 0, len(domains))
	for _, d := range domains {
		domainList = append(domainList, d.Domain)
	}

	// 如果没有可用域名，返回空数组
	if domainList == nil {
		domainList = []string{}
	}

	Success(c, gin.H{
		"domains": domainList,
		"count":   len(domainList),
	})
}

// GetSystemConfig godoc
// @Summary 获取系统配置
// @Description 获取前端需要的公开系统配置（公开接口，无需认证）
// @Tags Public
// @Produce json
// @Success 200 {object} Response{data=object{domains=[]string,defaultDomain=string,features=object}}
// @Router /v1/public/config [get]
func (h *PublicHandler) GetSystemConfig(c *gin.Context) {
	// 获取已激活的系统域名
	domains, err := h.systemDomainService.GetActiveDomains()
	if err != nil {
		InternalError(c, "获取系统配置失败")
		return
	}

	// 提取域名列表和默认域名
	domainList := make([]string, 0, len(domains))
	var defaultDomain string

	for _, d := range domains {
		domainList = append(domainList, d.Domain)
		if d.IsDefault {
			defaultDomain = d.Domain
		}
	}

	// 如果没有默认域名，使用第一个域名
	if defaultDomain == "" && len(domainList) > 0 {
		defaultDomain = domainList[0]
	}

	Success(c, gin.H{
		"domains":       domainList,
		"defaultDomain": defaultDomain,
		"features": gin.H{
			"websocket":   true,
			"attachments": true,
			"aliases":     true,
			"search":      true,
			"tags":        true,
			"webhooks":    true,
		},
	})
}
