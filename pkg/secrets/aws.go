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

package secrets

import (
	"context"
	"fmt"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

// AWSSecretsManagerStore AWS Secrets Manager 实现
type AWSSecretsManagerStore struct {
	client       *secretsmanager.Client
	region       string
	secretPrefix string
}

// AWSConfig AWS Secrets Manager 配置
type AWSConfig struct {
	Region       string // AWS 区域，如 "us-east-1"
	AccessKey    string // AWS Access Key（可选，使用环境变量或 IAM 角色）
	SecretKey    string // AWS Secret Key（可选）
	Endpoint     string // 自定义端点（可选，用于本地测试或私有端点）
	SecretPrefix string // Secret 名称前缀（如 "aetheris/"）
}

// NewAWSSecretsManagerStore 创建 AWS Secrets Manager Store
func NewAWSSecretsManagerStore(cfg AWSConfig) (Store, error) {
	var opts []func(*awsConfig.LoadOptions) error
	opts = append(opts, awsConfig.WithRegion(cfg.Region))

	// 使用静态凭证（如果提供）
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		opts = append(opts, awsConfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}

	// 加载 AWS 配置
	awsCfg, err := awsConfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// 创建 Secrets Manager 客户端
	client := secretsmanager.NewFromConfig(awsCfg)
	if cfg.Endpoint != "" {
		client = secretsmanager.NewFromConfig(awsCfg, func(o *secretsmanager.Options) {
			o.BaseEndpoint = &cfg.Endpoint
		})
	}

	return &AWSSecretsManagerStore{
		client:       client,
		region:       cfg.Region,
		secretPrefix: cfg.SecretPrefix,
	}, nil
}

// Get 获取 secret 值
func (s *AWSSecretsManagerStore) Get(ctx context.Context, key string) (string, error) {
	secretName := s.getSecretName(key)

	input := &secretsmanager.GetSecretValueInput{
		SecretId: &secretName,
	}

	result, err := s.client.GetSecretValue(ctx, input)
	if err != nil {
		var resourceNotFound *types.ResourceNotFoundException
		if err.As(&resourceNotFound) {
			return "", fmt.Errorf("secret %q not found", key)
		}
		return "", fmt.Errorf("failed to get secret %q: %w", key, err)
	}

	// 解码 secret
	if result.SecretString != nil {
		return *result.SecretString, nil
	}

	if result.SecretBinary != nil {
		return string(result.SecretBinary), nil
	}

	return "", fmt.Errorf("empty secret value for %q", key)
}

// Set 设置 secret 值
func (s *AWSSecretsManagerStore) Set(ctx context.Context, key string, value string) error {
	secretName := s.getSecretName(key)

	input := &secretsmanager.PutSecretValueInput{
		SecretId:     &secretName,
		SecretString: &value,
	}

	_, err := s.client.PutSecretValue(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to set secret %q: %w", key, err)
	}

	return nil
}

// Delete 删除 secret
func (s *AWSSecretsManagerStore) Delete(ctx context.Context, key string) error {
	secretName := s.getSecretName(key)

	input := &secretsmanager.DeleteSecretInput{
		SecretId: &secretName,
	}

	_, err := s.client.DeleteSecret(ctx, input)
	if err != nil {
		var resourceNotFound *types.ResourceNotFoundException
		if err.As(&resourceNotFound) {
			// Secret 不存在，视为删除成功
			return nil
		}
		return fmt.Errorf("failed to delete secret %q: %w", key, err)
	}

	return nil
}

// List 列出所有 secret keys
func (s *AWSSecretsManagerStore) List(ctx context.Context, prefix string) ([]string, error) {
	filter := s.getSecretName(prefix)

	input := &secretsmanager.ListSecretsInput{
		MaxResults: ptrInt32(100),
		Filters: []types.Filter{
			{
				Key:    types.FilterNameStringTypeName,
				Values: []string{filter},
			},
		},
	}

	var keys []string
	paginator := secretsmanager.NewListSecretsPaginator(s.client, input)

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}

		for _, secret := range output.SecretList {
			if secret.Name != nil {
				key := s.stripPrefix(*secret.Name)
				keys = append(keys, key)
			}
		}
	}

	return keys, nil
}

// getSecretName 获取完整的 secret 名称
func (s *AWSSecretsManagerStore) getSecretName(key string) string {
	if s.secretPrefix == "" {
		return key
	}
	return s.secretPrefix + key
}

// stripPrefix 去掉前缀
func (s *AWSSecretsManagerStore) stripPrefix(name string) string {
	if s.secretPrefix == "" {
		return name
	}
	return name[len(s.secretPrefix):]
}

// ptrInt32 返回 int32 指针
func ptrInt32(v int) *int32 {
	i := int32(v)
	return &i
}
