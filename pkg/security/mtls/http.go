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

package mtls

import (
	"crypto/tls"
	"fmt"
)

// HTTPConfig HTTP mTLS 配置
type HTTPConfig struct {
	Enabled            bool   // 是否启用 HTTPS + mTLS
	CertFile           string // 服务端证书路径
	KeyFile            string // 服务端私钥路径
	CAFile             string // CA 证书路径（用于验证客户端证书）
	ClientCertFile     string // 客户端证书路径（用于出站连接）
	ClientKeyFile      string // 客户端私钥路径
	InsecureSkipVerify bool   // 开发环境跳过证书验证
}

// ToTLSConfig 转换为标准 TLS 配置
func (c HTTPConfig) ToTLSConfig() (*tls.Config, error) {
	if !c.Enabled {
		return nil, nil
	}

	cfg := Config{
		Enabled:            c.Enabled,
		CertFile:           c.CertFile,
		KeyFile:            c.KeyFile,
		CAFile:             c.CAFile,
		ClientCertFile:     c.ClientCertFile,
		ClientKeyFile:      c.ClientKeyFile,
		InsecureSkipVerify: c.InsecureSkipVerify,
	}

	return ServerTLSConfig(cfg)
}

// HTTPClientConfig HTTP 客户端 TLS 配置
type HTTPClientConfig struct {
	Enabled            bool   // 是否启用 HTTPS
	CertFile           string // 客户端证书路径
	KeyFile            string // 客户端私钥路径
	CAFile             string // CA 证书路径（用于验证服务端证书）
	InsecureSkipVerify bool   // 开发环境跳过证书验证
}

// ToTLSConfig 转换为标准 TLS 配置
func (c HTTPClientConfig) ToTLSConfig() (*tls.Config, error) {
	if !c.Enabled {
		return nil, nil
	}

	cfg := Config{
		Enabled:            true,
		ClientCertFile:     c.CertFile,
		ClientKeyFile:      c.KeyFile,
		CAFile:             c.CAFile,
		InsecureSkipVerify: c.InsecureSkipVerify,
	}

	return ClientTLSConfig(cfg)
}

// NewHTTPConfig 从通用配置创建 HTTP 配置
func NewHTTPConfig(enabled bool, certFile, keyFile, caFile string) HTTPConfig {
	return HTTPConfig{
		Enabled:  enabled,
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
	}
}

// GetTLSConfig 返回用于 HTTP 服务的 TLS 配置
// 返回 (nil, nil) 表示不启用 TLS
func GetTLSConfig(enabled bool, certFile, keyFile, caFile string) (*tls.Config, error) {
	if !enabled {
		return nil, nil
	}

	if certFile == "" || keyFile == "" {
		return nil, fmt.Errorf("certificate and key files are required for TLS")
	}

	cfg := Config{
		Enabled:  true,
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
	}

	return ServerTLSConfig(cfg)
}

// GetClientTLSConfig 返回用于 HTTP 客户端的 TLS 配置
func GetClientTLSConfig(enabled bool, certFile, keyFile, caFile string, insecureSkipVerify bool) (*tls.Config, error) {
	if !enabled {
		return nil, nil
	}

	cfg := Config{
		Enabled:            true,
		ClientCertFile:     certFile,
		ClientKeyFile:      keyFile,
		CAFile:             caFile,
		InsecureSkipVerify: insecureSkipVerify,
	}

	return ClientTLSConfig(cfg)
}
