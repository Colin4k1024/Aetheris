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

package sso

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	dsig "github.com/russellhaering/goxmldsig"
	"golang.org/x/oauth2"
)

// fakeRand implements io.Reader for deterministic key generation in tests.
type fakeRand struct{}

func (f *fakeRand) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = byte(i)
	}
	return len(p), nil
}

// generateTestCert creates a self-signed RSA cert+key for SAML testing.
func generateTestCert(t *testing.T) (certPEM string, cert *x509.Certificate, privKey *rsa.PrivateKey) {
	t.Helper()
	privKey, err := rsa.GenerateKey(&fakeRand{}, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test-idp",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(&fakeRand{}, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	cert, err = x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}

	certPEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	}))

	return certPEM, cert, privKey
}

// signSAMLResponse creates a signed SAML response using the IdP's dsig signing context.
func signSAMLResponse(t *testing.T, sp *saml.ServiceProvider, cert *x509.Certificate, privKey *rsa.PrivateKey, attributes map[string]string) string {
	t.Helper()

	now := time.Now()
	_ = dsig.NewFakeClockAt(now) // ensure dsig clock is usable

	sessionExpiry := now.Add(8 * time.Hour)
	authnClassRef := &saml.AuthnContextClassRef{
		Value: "urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport",
	}

	assertion := saml.Assertion{
		ID:           "_test-assertion-id",
		IssueInstant: now,
		Version:      "2.0",
		Issuer: saml.Issuer{
			Format: "urn:oasis:names:tc:SAML:2.0:nameid-format:entity",
			Value:  "https://idp.example.com/sso",
		},
		Subject: &saml.Subject{
			NameID: &saml.NameID{
				Format: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
				Value:  "testuser@example.com",
			},
			SubjectConfirmations: []saml.SubjectConfirmation{
				{
					Method: "urn:oasis:names:tc:SAML:2.0:cm:bearer",
					SubjectConfirmationData: &saml.SubjectConfirmationData{
						NotOnOrAfter: now.Add(5 * time.Minute),
						Recipient:    sp.AcsURL.String(),
					},
				},
			},
		},
		Conditions: &saml.Conditions{
			AudienceRestrictions: []saml.AudienceRestriction{
				{
					Audience: saml.Audience{Value: sp.EntityID},
				},
			},
			NotBefore:    now.Add(-1 * time.Minute),
			NotOnOrAfter: now.Add(5 * time.Minute),
		},
		AuthnStatements: []saml.AuthnStatement{
			{
				AuthnInstant:        now,
				SessionIndex:        "_test-session",
				SessionNotOnOrAfter: &sessionExpiry,
				AuthnContext: saml.AuthnContext{
					AuthnContextClassRef: authnClassRef,
				},
			},
		},
	}

	if len(attributes) > 0 {
		stmt := saml.AttributeStatement{}
		for name, value := range attributes {
			stmt.Attributes = append(stmt.Attributes, saml.Attribute{
				Name: name,
				Values: []saml.AttributeValue{
					{Type: "xs:string", Value: value},
				},
			})
		}
		assertion.AttributeStatements = []saml.AttributeStatement{stmt}
	}

	response := saml.Response{
		ID:           "_test-response-id",
		InResponseTo: "",
		Destination:  sp.AcsURL.String(),
		IssueInstant: now,
		Version:      "2.0",
		Issuer: &saml.Issuer{
			Format: "urn:oasis:names:tc:SAML:2.0:nameid-format:entity",
			Value:  "https://idp.example.com/sso",
		},
		Status: saml.Status{
			StatusCode: saml.StatusCode{
				Value: saml.StatusSuccess,
			},
		},
	}

	// Build the response element with assertion
	responseEl := response.Element()
	assertionEl := assertion.Element()

	// Sign assertion using dsig signing context
	signingContext, err := dsig.NewSigningContext(privKey, [][]byte{cert.Raw})
	if err != nil {
		t.Fatalf("create signing context: %v", err)
	}
	signingContext.Canonicalizer = dsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")

	signedAssertionEl, err := signingContext.SignEnveloped(assertionEl)
	if err != nil {
		t.Fatalf("sign assertion: %v", err)
	}
	responseEl.AddChild(signedAssertionEl)

	// Sign response
	signedResponseEl, err := signingContext.SignEnveloped(responseEl)
	if err != nil {
		t.Fatalf("sign response: %v", err)
	}

	doc := etree.NewDocument()
	doc.SetRoot(signedResponseEl)
	xmlBytes, err := doc.WriteToBytes()
	if err != nil {
		t.Fatalf("write XML: %v", err)
	}

	return base64.StdEncoding.EncodeToString(xmlBytes)
}

// --- Configuration Tests ---

func TestNewSAMLClient_ValidConfig(t *testing.T) {
	certPEM, _, _ := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
		SLOURL:      "https://idp.example.com/slo",
	}

	client, err := NewSAMLClient(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.provider == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNewSAMLClient_MissingSSOURL(t *testing.T) {
	certPEM, _, _ := generateTestCert(t)
	config := SAMLConfig{
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
	}
	_, err := NewSAMLClient(config)
	if err == nil {
		t.Error("expected error for missing SSO URL")
	}
}

func TestNewSAMLClient_MissingCert(t *testing.T) {
	config := SAMLConfig{
		SSOURL:     "https://idp.example.com/sso",
		SPEntityID: "https://sp.example.com",
		ACSURL:     "https://sp.example.com/acs",
	}
	_, err := NewSAMLClient(config)
	if err == nil {
		t.Error("expected error for missing certificate")
	}
}

func TestNewSAMLClient_MissingSPEntityID(t *testing.T) {
	certPEM, _, _ := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		ACSURL:      "https://sp.example.com/acs",
	}
	_, err := NewSAMLClient(config)
	if err == nil {
		t.Error("expected error for missing SP entity ID")
	}
}

func TestNewSAMLClient_MissingACSURL(t *testing.T) {
	certPEM, _, _ := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
	}
	_, err := NewSAMLClient(config)
	if err == nil {
		t.Error("expected error for missing ACS URL")
	}
}

func TestNewSAMLClient_InvalidCert(t *testing.T) {
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: "not-a-valid-certificate",
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
	}
	_, err := NewSAMLClient(config)
	if err == nil {
		t.Error("expected error for invalid certificate")
	}
}

// --- LoginURL Tests ---

func TestSAMLClient_LoginURL(t *testing.T) {
	certPEM, _, _ := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
	}
	client, err := NewSAMLClient(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loginURL, err := client.LoginURL("relay-state-123", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loginURL == "" {
		t.Error("expected non-empty login URL")
	}
	if !strings.Contains(loginURL, "SAMLRequest=") {
		t.Errorf("expected SAMLRequest parameter in URL: %s", loginURL)
	}
	if !strings.Contains(loginURL, "https://idp.example.com/sso") {
		t.Errorf("expected IdP SSO URL in login URL: %s", loginURL)
	}
	if !strings.Contains(loginURL, "RelayState=relay-state-123") {
		t.Errorf("expected RelayState in URL: %s", loginURL)
	}
}

func TestSAMLClient_LoginURL_GeneratesRelayState(t *testing.T) {
	certPEM, _, _ := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
	}
	client, err := NewSAMLClient(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loginURL, err := client.LoginURL("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(loginURL, "RelayState=") {
		t.Errorf("expected auto-generated RelayState in URL: %s", loginURL)
	}
}

// --- Metadata Tests ---

func TestSAMLClient_SPMetadata(t *testing.T) {
	certPEM, _, _ := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
	}
	client, err := NewSAMLClient(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta, err := client.SPMetadata()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(meta) == 0 {
		t.Error("expected non-empty metadata")
	}
	if !strings.Contains(string(meta), "EntityDescriptor") {
		t.Error("expected EntityDescriptor in metadata")
	}
	if !strings.Contains(string(meta), "https://sp.example.com") {
		t.Error("expected SP entity ID in metadata")
	}
}

// --- Exchange Tests (rejection of invalid/unsigned responses) ---

func TestSAMLClient_Exchange_EmptyResponse(t *testing.T) {
	certPEM, _, _ := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
	}
	client, _ := NewSAMLClient(config)

	_, err := client.Exchange(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty SAML response")
	}
}

func TestSAMLClient_Exchange_InvalidBase64(t *testing.T) {
	certPEM, _, _ := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
	}
	client, _ := NewSAMLClient(config)

	_, err := client.Exchange(context.Background(), "!!!not-valid-base64-!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestSAMLClient_Exchange_InvalidXML(t *testing.T) {
	certPEM, _, _ := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
	}
	client, _ := NewSAMLClient(config)

	encoded := base64.StdEncoding.EncodeToString([]byte("not valid xml"))
	_, err := client.Exchange(context.Background(), encoded)
	if err == nil {
		t.Error("expected error for invalid XML")
	}
}

func TestSAMLClient_Exchange_UnsignedResponse(t *testing.T) {
	certPEM, _, _ := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
	}
	client, _ := NewSAMLClient(config)

	// Build an unsigned SAML response
	now := time.Now()
	authnClassRef := &saml.AuthnContextClassRef{
		Value: "urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport",
	}
	assertion := saml.Assertion{
		ID:           "_test-assertion",
		IssueInstant: now,
		Version:      "2.0",
		Issuer:       saml.Issuer{Value: "https://idp.example.com/sso"},
		Subject: &saml.Subject{
			NameID: &saml.NameID{Value: "test@example.com"},
		},
		Conditions: &saml.Conditions{
			AudienceRestrictions: []saml.AudienceRestriction{
				{Audience: saml.Audience{Value: "https://sp.example.com"}},
			},
		},
		AuthnStatements: []saml.AuthnStatement{
			{
				AuthnInstant: now,
				AuthnContext: saml.AuthnContext{
					AuthnContextClassRef: authnClassRef,
				},
			},
		},
	}

	response := saml.Response{
		ID:           "_test-response",
		InResponseTo: "",
		Destination:  "https://sp.example.com/acs",
		IssueInstant: now,
		Version:      "2.0",
		Issuer:       &saml.Issuer{Value: "https://idp.example.com/sso"},
		Status: saml.Status{
			StatusCode: saml.StatusCode{Value: saml.StatusSuccess},
		},
	}

	responseEl := response.Element()
	responseEl.AddChild(assertion.Element())
	doc := etree.NewDocument()
	doc.SetRoot(responseEl)
	xmlBytes, _ := doc.WriteToBytes()
	encoded := base64.StdEncoding.EncodeToString(xmlBytes)

	_, err := client.Exchange(context.Background(), encoded)
	if err == nil {
		t.Error("expected error for unsigned SAML response")
	}
}

func TestSAMLClient_Exchange_SignedResponse_Success(t *testing.T) {
	certPEM, cert, privKey := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
		AttributeMapping: map[string]string{
			"email":       "email",
			"displayName": "name",
		},
	}
	client, err := NewSAMLClient(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	encoded := signSAMLResponse(t, client.provider.sp, cert, privKey, map[string]string{
		"email":       "testuser@example.com",
		"displayName": "Test User",
	})

	token, err := client.Exchange(context.Background(), encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == nil {
		t.Fatal("expected non-nil token")
	}
	if token.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if token.TokenType != "SAML" {
		t.Errorf("expected token type SAML, got %s", token.TokenType)
	}

	// Verify we can extract user info
	userInfo, err := client.GetUserInfo(context.Background(), token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userInfo.Email != "testuser@example.com" {
		t.Errorf("email mismatch: %s", userInfo.Email)
	}
	if userInfo.Name != "Test User" {
		t.Errorf("name mismatch: %s", userInfo.Name)
	}
	if userInfo.Sub != "testuser@example.com" {
		t.Errorf("sub mismatch: %s", userInfo.Sub)
	}
}

func TestSAMLClient_Exchange_WrongAudience(t *testing.T) {
	certPEM, cert, privKey := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
	}
	client, _ := NewSAMLClient(config)

	// Build a signed response with wrong audience
	now := time.Now()
	assertion := saml.Assertion{
		ID:           "_test-aud-assertion",
		IssueInstant: now,
		Version:      "2.0",
		Issuer:       saml.Issuer{Value: "https://idp.example.com/sso"},
		Subject: &saml.Subject{
			NameID: &saml.NameID{Value: "test@example.com"},
		},
		Conditions: &saml.Conditions{
			AudienceRestrictions: []saml.AudienceRestriction{
				{Audience: saml.Audience{Value: "https://wrong-sp.example.com"}},
			},
		},
		AuthnStatements: []saml.AuthnStatement{
			{
				AuthnInstant: now,
				AuthnContext: saml.AuthnContext{
					AuthnContextClassRef: &saml.AuthnContextClassRef{
						Value: "urn:oasis:names:tc:SAML:2.0:ac:classes:Password",
					},
				},
			},
		},
	}

	response := saml.Response{
		ID:           "_test-aud-response",
		Destination:  "https://sp.example.com/acs",
		IssueInstant: now,
		Version:      "2.0",
		Issuer:       &saml.Issuer{Value: "https://idp.example.com/sso"},
		Status:       saml.Status{StatusCode: saml.StatusCode{Value: saml.StatusSuccess}},
	}

	responseEl := response.Element()
	assertionEl := assertion.Element()

	signingContext, err := dsig.NewSigningContext(privKey, [][]byte{cert.Raw})
	if err != nil {
		t.Fatalf("create signing context: %v", err)
	}
	signingContext.Canonicalizer = dsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")

	signedAssertionEl, err := signingContext.SignEnveloped(assertionEl)
	if err != nil {
		t.Fatalf("sign assertion: %v", err)
	}
	responseEl.AddChild(signedAssertionEl)

	signedResponseEl, err := signingContext.SignEnveloped(responseEl)
	if err != nil {
		t.Fatalf("sign response: %v", err)
	}

	doc := etree.NewDocument()
	doc.SetRoot(signedResponseEl)
	xmlBytes, _ := doc.WriteToBytes()
	encoded := base64.StdEncoding.EncodeToString(xmlBytes)

	_, err = client.Exchange(context.Background(), encoded)
	if err == nil {
		t.Error("expected error for wrong audience")
	}
}

func TestSAMLClient_Exchange_ReplayDetection(t *testing.T) {
	certPEM, cert, privKey := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
	}
	client, _ := NewSAMLClient(config)

	encoded := signSAMLResponse(t, client.provider.sp, cert, privKey, nil)

	// First exchange should succeed
	_, err := client.Exchange(context.Background(), encoded)
	if err != nil {
		t.Fatalf("first exchange should succeed: %v", err)
	}

	// Second exchange with same response should fail (replay)
	_, err = client.Exchange(context.Background(), encoded)
	if err == nil {
		t.Error("expected replay detection error on second exchange")
	}
}

// --- GetUserInfo Tests ---

func TestSAMLClient_GetUserInfo_NoSession(t *testing.T) {
	certPEM, _, _ := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
	}
	client, _ := NewSAMLClient(config)

	_, err := client.GetUserInfo(context.Background(), &oauth2.Token{AccessToken: "nonexistent"})
	if err == nil {
		t.Error("expected error for missing session")
	}
}

func TestSAMLClient_GetUserInfo_NilToken(t *testing.T) {
	certPEM, _, _ := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
	}
	client, _ := NewSAMLClient(config)

	_, err := client.GetUserInfo(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil token")
	}
}

// --- ValidateUser Tests ---

func TestSAMLClient_ValidateUser_Nil(t *testing.T) {
	certPEM, _, _ := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
	}
	client, _ := NewSAMLClient(config)

	err := client.ValidateUser(nil)
	if err == nil {
		t.Error("expected error for nil user info")
	}
}

func TestSAMLClient_ValidateUser_NoEmail(t *testing.T) {
	certPEM, _, _ := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
	}
	client, _ := NewSAMLClient(config)

	err := client.ValidateUser(&UserInfo{Name: "No Email"})
	if err == nil {
		t.Error("expected error for missing email")
	}
}

func TestSAMLClient_ValidateUser_WithEmail(t *testing.T) {
	certPEM, _, _ := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
	}
	client, _ := NewSAMLClient(config)

	err := client.ValidateUser(&UserInfo{Email: "user@example.com"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- GetLogoutURL Tests ---

func TestSAMLClient_GetLogoutURL_NoSLO(t *testing.T) {
	certPEM, _, _ := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
	}
	client, _ := NewSAMLClient(config)

	logoutURL := client.GetLogoutURL("https://app.com/logout")
	if logoutURL != "https://app.com/logout" {
		t.Errorf("expected redirect URL when no SLO configured, got: %s", logoutURL)
	}
}

func TestSAMLClient_GetLogoutURL_WithSLO(t *testing.T) {
	certPEM, _, _ := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
		SLOURL:      "https://idp.example.com/slo",
	}
	client, _ := NewSAMLClient(config)

	logoutURL := client.GetLogoutURL("https://app.com/logout")
	if logoutURL == "" {
		t.Error("expected non-empty logout URL")
	}
	if !strings.Contains(logoutURL, "https://idp.example.com/slo") {
		t.Errorf("expected IdP SLO URL in logout URL: %s", logoutURL)
	}
}

// --- Replay Detection Tests ---

func TestSAMLReplayDetection(t *testing.T) {
	certPEM, _, _ := generateTestCert(t)
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: certPEM,
		SPEntityID:  "https://sp.example.com",
		ACSURL:      "https://sp.example.com/acs",
	}
	client, err := NewSAMLClient(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prov := client.provider

	if prov.isReplay("req-123") {
		t.Error("expected first use to not be replay")
	}
	if !prov.isReplay("req-123") {
		t.Error("expected second use to be detected as replay")
	}
	if prov.isReplay("req-456") {
		t.Error("expected different ID to not be replay")
	}
}

// --- Certificate Parsing Tests ---

func TestParseSAMLCertificate_PEM(t *testing.T) {
	certPEM, expectedCert, _ := generateTestCert(t)
	cert, err := parseSAMLCertificate(certPEM)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cert.Subject.CommonName != expectedCert.Subject.CommonName {
		t.Errorf("cert subject mismatch: %s vs %s", cert.Subject.CommonName, expectedCert.Subject.CommonName)
	}
}

func TestParseSAMLCertificate_Invalid(t *testing.T) {
	_, err := parseSAMLCertificate("totally invalid")
	if err == nil {
		t.Error("expected error for invalid certificate")
	}
}

// --- Attribute Extraction Tests ---

func TestExtractUserInfoFromAssertion(t *testing.T) {
	assertion := &saml.Assertion{
		Subject: &saml.Subject{
			NameID: &saml.NameID{
				Format: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
				Value:  "user@example.com",
			},
		},
		AttributeStatements: []saml.AttributeStatement{
			{
				Attributes: []saml.Attribute{
					{
						Name: "email",
						Values: []saml.AttributeValue{
							{Value: "user@example.com"},
						},
					},
					{
						Name: "displayName",
						Values: []saml.AttributeValue{
							{Value: "Test User"},
						},
					},
				},
			},
		},
	}

	mapping := map[string]string{
		"email":       "email",
		"displayName": "name",
	}

	info := extractUserInfoFromAssertion(assertion, mapping)
	if info.Email != "user@example.com" {
		t.Errorf("email mismatch: %s", info.Email)
	}
	if info.Name != "Test User" {
		t.Errorf("name mismatch: %s", info.Name)
	}
	if info.Sub != "user@example.com" {
		t.Errorf("sub mismatch: %s", info.Sub)
	}
	if len(info.Claims) != 2 {
		t.Errorf("expected 2 claims, got %d", len(info.Claims))
	}
}

func TestExtractUserInfoFromAssertion_DefaultMapping(t *testing.T) {
	assertion := &saml.Assertion{
		AttributeStatements: []saml.AttributeStatement{
			{
				Attributes: []saml.Attribute{
					{
						Name: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
						Values: []saml.AttributeValue{
							{Value: "default@example.com"},
						},
					},
				},
			},
		},
	}

	info := extractUserInfoFromAssertion(assertion, nil)
	if info.Email != "default@example.com" {
		t.Errorf("expected default email mapping, got: %s", info.Email)
	}
}

func TestExtractUserInfoFromAssertion_Nil(t *testing.T) {
	info := extractUserInfoFromAssertion(nil, nil)
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.Claims == nil {
		t.Error("expected non-nil claims map")
	}
}

func TestExtractUserInfoFromAssertion_NameIDAsEmail(t *testing.T) {
	assertion := &saml.Assertion{
		Subject: &saml.Subject{
			NameID: &saml.NameID{
				Value: "user@example.com",
			},
		},
	}

	info := extractUserInfoFromAssertion(assertion, nil)
	if info.Email != "user@example.com" {
		t.Errorf("expected NameID used as email: %s", info.Email)
	}
	if info.Sub != "user@example.com" {
		t.Errorf("expected NameID as sub: %s", info.Sub)
	}
}
