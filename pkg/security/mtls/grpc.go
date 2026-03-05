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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCServerOption 创建 gRPC 服务端的 mTLS 选项
func GRPCServerOption(cfg Config) (grpc.ServerOption, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	tlsConfig, err := ServerTLSConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create server TLS config: %w", err)
	}

	creds := credentials.NewTLS(tlsConfig)
	return grpc.Creds(creds), nil
}

// GRPCDialOption 创建 gRPC 客户端的 mTLS 连接选项
func GRPCDialOption(cfg Config) (grpc.DialOption, error) {
	if !cfg.Enabled {
		return grpc.WithTransportCredentials(insecure.NewCredentials()), nil
	}

	tlsConfig, err := ClientTLSConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create client TLS config: %w", err)
	}

	creds := credentials.NewTLS(tlsConfig)
	return grpc.WithTransportCredentials(creds), nil
}

// GRPCServerTLSConfig 转换为 grpc/credentials 使用的配置
func GRPCServerTLSConfig(cfg Config) (*tls.Config, error) {
	return ServerTLSConfig(cfg)
}

// GRPCClientTLSConfig 转换为 grpc/credentials 使用的配置
func GRPCClientTLSConfig(cfg Config) (*tls.Config, error) {
	return ClientTLSConfig(cfg)
}
