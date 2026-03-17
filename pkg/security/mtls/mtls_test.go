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
	"testing"
)

func TestConfig(t *testing.T) {
	cfg := Config{
		Enabled:            true,
		CertFile:           "/path/to/cert.pem",
		KeyFile:            "/path/to/key.pem",
		CAFile:             "/path/to/ca.pem",
		ClientCertFile:     "/path/to/client-cert.pem",
		ClientKeyFile:      "/path/to/client-key.pem",
		InsecureSkipVerify: true,
	}

	if !cfg.Enabled {
		t.Error("expected Enabled to be true")
	}
	if cfg.CertFile != "/path/to/cert.pem" {
		t.Errorf("expected /path/to/cert.pem, got %s", cfg.CertFile)
	}
	if cfg.KeyFile != "/path/to/key.pem" {
		t.Errorf("expected /path/to/key.pem, got %s", cfg.KeyFile)
	}
	if cfg.CAFile != "/path/to/ca.pem" {
		t.Errorf("expected /path/to/ca.pem, got %s", cfg.CAFile)
	}
	if !cfg.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify to be true")
	}
}

func TestConfig_Disabled(t *testing.T) {
	cfg := Config{
		Enabled: false,
	}

	if cfg.Enabled {
		t.Error("expected Enabled to be false")
	}
}

func TestServerTLSConfig_Disabled(t *testing.T) {
	cfg := Config{
		Enabled: false,
	}

	tlsConfig, err := ServerTLSConfig(cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if tlsConfig != nil {
		t.Error("expected nil tls config when disabled")
	}
}

func TestClientTLSConfig_Disabled(t *testing.T) {
	cfg := Config{
		Enabled: false,
	}

	tlsConfig, err := ClientTLSConfig(cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if tlsConfig != nil {
		t.Error("expected nil tls config when disabled")
	}
}

func TestClientTLSConfig_WithoutClientCert(t *testing.T) {
	cfg := Config{
		Enabled:            true,
		CertFile:           "/path/to/cert.pem",
		KeyFile:            "/path/to/key.pem",
		InsecureSkipVerify: true,
		CAFile:             "/path/to/ca.pem", // Add CA file to trigger cert pool creation
	}

	// This will fail because cert files don't exist, but tests the code path
	_, err := ClientTLSConfig(cfg)
	if err == nil {
		t.Error("expected error for missing CA file")
	}
}
