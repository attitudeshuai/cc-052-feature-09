// Package region 提供扫码地区的服务端解析。
//
// 扫码地区只能由服务端根据真实对端 IP 解析得出，绝不采信
// X-Forwarded-For 等客户端可伪造的请求来源。解析结果存行政区划代码，
// 不落原始 IP（IP 属于个人信息）。
package region

import (
	"net"
	"strings"
)

// Resolver 按配置的 CIDR→地区码映射解析 IP 所属地区。
type Resolver struct {
	entries []entry
}

type entry struct {
	cidr   *net.IPNet
	region string
}

// NewResolver 解析配置串，格式为 "cidr=地区码;cidr=地区码;..."，
// 例如 "203.0.113.0/24=110105;198.51.100.0/24=310104"。
// 空配置或非法条目会被跳过——解析不到时地区一律为未知，而不是退回去相信请求头。
func NewResolver(mapping string) *Resolver {
	r := &Resolver{}
	for _, part := range strings.Split(mapping, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		cidr, region, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		region = strings.TrimSpace(region)
		_, ipNet, err := net.ParseCIDR(strings.TrimSpace(cidr))
		if err != nil || region == "" {
			continue
		}
		r.entries = append(r.entries, entry{cidr: ipNet, region: region})
	}
	return r
}

// Resolve 返回 ip 对应的地区码；无法确定时 ok=false。
func (r *Resolver) Resolve(ip string) (region string, ok bool) {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return "", false
	}
	for _, e := range r.entries {
		if e.cidr.Contains(parsed) {
			return e.region, true
		}
	}
	return "", false
}
