// Package regioncode 校验扫码端显式上报的行政区划码（GB/T 2260，6 位数字）。
// 扫码地区只能来自客户端定位授权后上报的区划码，
// 不能用 X-Forwarded-For / 对端 IP 等请求来源信息推断或冒充。
package regioncode

import "regexp"

var regionCodePattern = regexp.MustCompile(`^\d{6}$`)

// 省级行政区划代码前两位（GB/T 2260），用于剔除 000000/999999 之类格式合法但不存在的码
var validProvincePrefix = map[string]bool{
	"11": true, "12": true, "13": true, "14": true, "15": true,
	"21": true, "22": true, "23": true,
	"31": true, "32": true, "33": true, "34": true, "35": true, "36": true, "37": true,
	"41": true, "42": true, "43": true, "44": true, "45": true, "46": true,
	"50": true, "51": true, "52": true, "53": true, "54": true,
	"61": true, "62": true, "63": true, "64": true, "65": true,
	"71": true, "81": true, "82": true,
}

// Valid 判断是否为合法的 6 位行政区划码（省级前缀存在）。
func Valid(code string) bool {
	if !regionCodePattern.MatchString(code) {
		return false
	}
	return validProvincePrefix[code[:2]]
}
