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

// Package mtls provides mTLS (mutual TLS) support for secure communication.
package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// Config mTLS 配置
type Config struct {
	Enabled            bool   // 是否启用 mTLS
	CertFile           string // 服务端证书路径
	KeyFile            string // 服务端私钥路径
	CAFile             string // CA 证书路径（用于验证客户端证书）
	ClientCertFile     string // 客户端证书路径（用于出站连接）
	ClientKeyFile      string // 客户端私钥路径
	InsecureSkipVerify bool   // 开发环境跳过证书验证
}

// ServerTLSConfig 创建服务端的 TLS 配置
func ServerTLSConfig(cfg Config) (*tls.Config, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	// 加载服务端证书
	serverCert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	// 加载 CA 证书（用于验证客户端）
	var clientCertPool *x509.CertPool
	if cfg.CAFile != "" {
		caCert, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		clientCertPool = x509.NewCertPool()
		if !clientCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    clientCertPool,
		ClientAuth:   tls.RequestClientCert, // 强制要求客户端证书
		MinVersion:   tls.VersionTLS12,
	}

	return tlsConfig, nil
}

// ClientTLSConfig 创建客户端的 TLS 配置
func ClientTLSConfig(cfg Config) (*tls.Config, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	// 加载客户端证书
	var certificates []tls.Certificate
	if cfg.ClientCertFile != "" && cfg.ClientKeyFile != "" {
		clientCert, err := tls.LoadX509KeyPair(cfg.ClientCertFile, cfg.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		certificates = append(certificates, clientCert)
	}

	// 加载 CA 证书（用于验证服务端）
	var serverCertPool *x509.CertPool
	if cfg.CAFile != "" {
		caCert, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		serverCertPool = x509.NewCertPool()
		if !serverCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
	}

	tlsConfig := &tls.Config{
		Certificates:       certificates,
		RootCAs:            serverCertPool,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}

	return tlsConfig, nil
}
