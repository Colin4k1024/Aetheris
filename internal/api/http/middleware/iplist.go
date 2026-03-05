// Copyright 2026 fanjia1024
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package middleware

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

var errInvalidIP = errors.New("invalid IP address")

// IPAllowList IP 白名单中间件
type IPAllowList struct {
	allowList      []net.IPNet
	blockList      []net.IPNet
	trustedProxies []net.IPNet
}

// NewIPAllowList 创建 IP 白名单中间件
// allowIPs: 允许的 IP/CIDR 列表
// blockIPs: 禁止的 IP/CIDR 列表（优先级高于 allowList）
// trustedProxies: 受信任的代理服务器 IP（用于解析 X-Forwarded-For）
func NewIPAllowList(allowIPs, blockIPs, trustedProxies []string) (*IPAllowList, error) {
	allowList, err := parseIPList(allowIPs)
	if err != nil {
		return nil, err
	}

	blockList, err := parseIPList(blockIPs)
	if err != nil {
		return nil, err
	}

	proxies, err := parseIPList(trustedProxies)
	if err != nil {
		return nil, err
	}

	return &IPAllowList{
		allowList:      allowList,
		blockList:      blockList,
		trustedProxies: proxies,
	}, nil
}

// parseIPList 解析 IP/CIDR 列表
func parseIPList(ips []string) ([]net.IPNet, error) {
	var result []net.IPNet
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}

		// 判断是否为 CIDR 格式
		if strings.Contains(ip, "/") {
			_, network, err := net.ParseCIDR(ip)
			if err != nil {
				return nil, err
			}
			result = append(result, *network)
		} else {
			parsedIP := net.ParseIP(ip)
			if parsedIP == nil {
				return nil, errInvalidIP
			}
			// 转换为 /32 或 /128 CIDR
			if parsedIP.To4() != nil {
				result = append(result, net.IPNet{
					IP:   parsedIP,
					Mask: net.IPv4Mask(255, 255, 255, 255),
				})
			} else {
				result = append(result, net.IPNet{
					IP:   parsedIP,
					Mask: net.CIDRMask(128, 128),
				})
			}
		}
	}
	return result, nil
}

// getClientIP 获取客户端真实 IP
func (i *IPAllowList) getClientIP(xForwardedFor string) string {
	// 如果配置了受信任的代理，解析 X-Forwarded-For
	if len(i.trustedProxies) > 0 && xForwardedFor != "" {
		// X-Forwarded-For 可能包含多个 IP，第一个是非代理的客户端 IP
		ips := strings.Split(xForwardedFor, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip == "" {
				continue
			}
			parsedIP := net.ParseIP(ip)
			if parsedIP == nil {
				continue
			}
			// 检查是否来自受信任的代理
			isTrusted := false
			for _, proxy := range i.trustedProxies {
				if proxy.Contains(parsedIP) {
					isTrusted = true
					break
				}
			}
			if isTrusted {
				continue
			}
			// 找到第一个非代理 IP
			return ip
		}
	}
	return ""
}

// isIPAllowed 检查 IP 是否在白名单中
func (i *IPAllowList) isIPAllowed(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// 先检查黑名单
	for _, block := range i.blockList {
		if block.Contains(ip) {
			hlog.CtxWarnf(context.Background(), "IP %s blocked by blacklist", ipStr)
			return false
		}
	}

	// 白名单为空表示允许所有
	if len(i.allowList) == 0 {
		return true
	}

	// 检查白名单
	for _, allow := range i.allowList {
		if allow.Contains(ip) {
			return true
		}
	}

	return false
}

// Middleware 返回 Hertz 中间件
func (i *IPAllowList) Middleware() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		// 获取客户端 IP
		clientIP := ctx.ClientIP()
		if clientIP == "" {
			ctx.AbortWithStatus(consts.StatusForbidden)
			return
		}

		// 尝试从 X-Forwarded-For 获取真实 IP
		xForwardedFor := string(ctx.Request.Header.Peek("X-Forwarded-For"))
		if realIP := i.getClientIP(xForwardedFor); realIP != "" {
			clientIP = realIP
		}

		// 检查 IP 是否允许
		if !i.isIPAllowed(clientIP) {
			hlog.CtxWarnf(c, "IP %s denied by allowlist", clientIP)
			ctx.JSON(consts.StatusForbidden, map[string]interface{}{
				"code":    consts.StatusForbidden,
				"message": "access denied: your IP is not in the allowlist",
			})
			ctx.Abort()
			return
		}

		ctx.Next(c)
	}
}

// IPAllowListConfig IP 白名单配置
type IPAllowListConfig struct {
	Enabled        bool
	AllowIPs       []string
	BlockIPs       []string
	TrustedProxies []string
}

// NewIPAllowListFromConfig 从配置创建 IP 白名单
func NewIPAllowListFromConfig(cfg IPAllowListConfig) (*IPAllowList, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	return NewIPAllowList(cfg.AllowIPs, cfg.BlockIPs, cfg.TrustedProxies)
}
