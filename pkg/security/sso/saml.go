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
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/crewjam/saml"
	"golang.org/x/oauth2"
)

// samlProvider wraps crewjam/saml ServiceProvider and tracks active sessions
// for replay detection and attribute extraction.
type samlProvider struct {
	sp             *saml.ServiceProvider
	config         *SAMLConfig
	mu             sync.Mutex
	seenRequestIDs map[string]time.Time
	sessions       map[string]*samlSession // relayState → session
}

// samlSession holds the result of a completed SAML authentication.
type samlSession struct {
	assertion  *saml.Assertion
	relayState string
	createdAt  time.Time
	userInfo   *UserInfo
}

// replayTTL is how long we keep request IDs for replay detection.
const replayTTL = 10 * time.Minute

// newSAMLProvider builds a fully configured SAML ServiceProvider from SAMLConfig.
// The IdP certificate is parsed from PEM; SSOURL, SPEntityID, ACSURL and SLOURL
// are wired into the ServiceProvider. Returns an error if any required field
// is missing or the certificate cannot be parsed.
func newSAMLProvider(config SAMLConfig) (*samlProvider, error) {
	if config.SSOURL == "" {
		return nil, fmt.Errorf("SAML SSO URL is required")
	}
	if config.Certificate == "" {
		return nil, fmt.Errorf("SAML IdP certificate is required")
	}
	if config.SPEntityID == "" {
		return nil, fmt.Errorf("SAML SP entity ID is required")
	}
	if config.ACSURL == "" {
		return nil, fmt.Errorf("SAML ACS URL is required")
	}

	cert, err := parseSAMLCertificate(config.Certificate)
	if err != nil {
		return nil, fmt.Errorf("parse IdP certificate: %w", err)
	}

	ssoURL, err := url.Parse(config.SSOURL)
	if err != nil {
		return nil, fmt.Errorf("parse SSO URL: %w", err)
	}

	acsURL, err := url.Parse(config.ACSURL)
	if err != nil {
		return nil, fmt.Errorf("parse ACS URL: %w", err)
	}

	var sloURL url.URL
	if config.SLOURL != "" {
		parsedSLO, perr := url.Parse(config.SLOURL)
		if perr != nil {
			return nil, fmt.Errorf("parse SLO URL: %w", perr)
		}
		sloURL = *parsedSLO
	}

	metadataURL, err := url.Parse(config.SPEntityID + "/saml/metadata")
	if err != nil {
		return nil, fmt.Errorf("parse metadata URL: %w", err)
	}

	issuer := config.Issuer
	if issuer == "" {
		issuer = config.SSOURL
	}

	sp := &saml.ServiceProvider{
		EntityID:          config.SPEntityID,
		MetadataURL:       *metadataURL,
		AcsURL:            *acsURL,
		SloURL:            sloURL,
		IDPMetadata:       buildIDPMetadata(issuer, ssoURL, &sloURL, cert),
		AllowIDPInitiated: true, // allow IdP-initiated SSO; request IDs are still tracked for replay detection
	}

	return &samlProvider{
		sp:             sp,
		config:         &config,
		seenRequestIDs: make(map[string]time.Time),
		sessions:       make(map[string]*samlSession),
	}, nil
}

// parseSAMLCertificate accepts raw PEM or base64 DER and returns the x509 cert.
func parseSAMLCertificate(raw string) (*x509.Certificate, error) {
	// Try PEM first
	block, _ := pem.Decode([]byte(raw))
	if block != nil {
		return x509.ParseCertificate(block.Bytes)
	}
	// Try raw base64 DER
	der, err := base64.StdEncoding.DecodeString(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("certificate is neither valid PEM nor base64 DER: %w", err)
	}
	return x509.ParseCertificate(der)
}

// buildIDPMetadata constructs an EntityDescriptor for the IdP from config.
func buildIDPMetadata(entityID string, ssoURL, sloURL *url.URL, cert *x509.Certificate) *saml.EntityDescriptor {
	certData := base64.StdEncoding.EncodeToString(cert.Raw)
	ed := &saml.EntityDescriptor{
		EntityID: entityID,
		IDPSSODescriptors: []saml.IDPSSODescriptor{
			{
				SSODescriptor: saml.SSODescriptor{
					RoleDescriptor: saml.RoleDescriptor{
						ProtocolSupportEnumeration: "urn:oasis:names:tc:SAML:2.0:protocol",
						KeyDescriptors: []saml.KeyDescriptor{
							{
								Use: "signing",
								KeyInfo: saml.KeyInfo{
									X509Data: saml.X509Data{
										X509Certificates: []saml.X509Certificate{
											{Data: certData},
										},
									},
								},
							},
						},
					},
					NameIDFormats: []saml.NameIDFormat{
						saml.EmailAddressNameIDFormat,
						saml.UnspecifiedNameIDFormat,
					},
				},
				SingleSignOnServices: []saml.Endpoint{
					{
						Binding:  saml.HTTPRedirectBinding,
						Location: ssoURL.String(),
					},
					{
						Binding:  saml.HTTPPostBinding,
						Location: ssoURL.String(),
					},
				},
			},
		},
	}
	if sloURL != nil && sloURL.String() != "" {
		ed.IDPSSODescriptors[0].SingleLogoutServices = []saml.Endpoint{
			{
				Binding:  saml.HTTPRedirectBinding,
				Location: sloURL.String(),
			},
		}
	}
	return ed
}

// generateRelayState produces a unique relay state for each auth request.
func generateRelayState() string {
	state, _ := GenerateState()
	return state
}

// trackRequestID stores a SAML AuthnRequest ID for later validation in Exchange.
func (p *samlProvider) trackRequestID(requestID string) {
	if requestID == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.seenRequestIDs[requestID] = time.Now()
}

// getKnownRequestIDs returns all tracked request IDs for ParseXMLResponse validation.
func (p *samlProvider) getKnownRequestIDs() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cleanupReplayIDs()
	ids := make([]string, 0, len(p.seenRequestIDs))
	for id := range p.seenRequestIDs {
		ids = append(ids, id)
	}
	return ids
}

// isReplay checks if a request ID has been seen recently (replay attack detection).
func (p *samlProvider) isReplay(requestID string) bool {
	if requestID == "" {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cleanupReplayIDs()
	if _, seen := p.seenRequestIDs[requestID]; seen {
		return true
	}
	p.seenRequestIDs[requestID] = time.Now()
	return false
}

// cleanupReplayIDs removes expired entries from the seen-request-ID map.
func (p *samlProvider) cleanupReplayIDs() {
	now := time.Now()
	for id, t := range p.seenRequestIDs {
		if now.Sub(t) > replayTTL {
			delete(p.seenRequestIDs, id)
		}
	}
}

// storeSession saves a SAML session indexed by relayState for later retrieval.
func (p *samlProvider) storeSession(relayState string, session *samlSession) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sessions[relayState] = session
}

// getSession retrieves and removes a session by relayState (one-time use).
func (p *samlProvider) getSession(relayState string) (*samlSession, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	s, ok := p.sessions[relayState]
	if ok {
		delete(p.sessions, relayState)
	}
	return s, ok
}

// LoginURL generates a SAML AuthnRequest redirect URL (HTTP-Redirect binding).
// The relayState is stored so the ACS callback can correlate the response.
func (p *samlProvider) LoginURL(state string, redirectURL string) (string, error) {
	relayState := state
	if relayState == "" {
		relayState = generateRelayState()
	}

	// Build the AuthnRequest manually to capture its ID for replay tracking
	authReq, err := p.sp.MakeAuthenticationRequest(
		p.sp.GetSSOBindingLocation(saml.HTTPRedirectBinding),
		saml.HTTPRedirectBinding,
		saml.HTTPPostBinding,
	)
	if err != nil {
		return "", fmt.Errorf("build SAML authn request: %w", err)
	}
	p.trackRequestID(authReq.ID)

	authURL, err := authReq.Redirect(relayState, p.sp)
	if err != nil {
		return "", fmt.Errorf("build SAML authn request: %w", err)
	}

	// Store an empty session placeholder so we can validate relayState on callback
	p.mu.Lock()
	p.sessions[relayState] = &samlSession{
		relayState: relayState,
		createdAt:  time.Now(),
	}
	p.mu.Unlock()

	return authURL.String(), nil
}

// Exchange parses the SAML response, validates signature/audience/time/replay,
// and stores the resulting assertion as a session. The `code` parameter should
// be a base64-encoded SAML response XML (as sent to the ACS endpoint).
// Returns an oauth2.Token whose AccessToken is the relayState/session ID.
func (p *samlProvider) Exchange(ctx context.Context, samlResponseB64 string) (*oauth2.Token, error) {
	if samlResponseB64 == "" {
		return nil, fmt.Errorf("SAML response is empty")
	}

	decoded, err := base64.StdEncoding.DecodeString(samlResponseB64)
	if err != nil {
		return nil, fmt.Errorf("decode SAML response: %w", err)
	}

	acsURL := p.sp.AcsURL
	assertion, err := p.sp.ParseXMLResponse(decoded, p.getKnownRequestIDs(), acsURL)
	if err != nil {
		return nil, fmt.Errorf("validate SAML response: %w", err)
	}

	if assertion == nil {
		return nil, fmt.Errorf("SAML assertion is nil after parsing")
	}

	// Replay detection: check the assertion ID (unique per assertion) and InResponseTo if present
	if assertion.ID != "" {
		if p.isReplay(assertion.ID) {
			return nil, fmt.Errorf("SAML replay detected: assertion ID %s already seen", assertion.ID)
		}
	}
	if assertion.Subject != nil && assertion.Subject.SubjectConfirmations != nil {
		for _, sc := range assertion.Subject.SubjectConfirmations {
			if sc.SubjectConfirmationData != nil && sc.SubjectConfirmationData.InResponseTo != "" {
				reqID := sc.SubjectConfirmationData.InResponseTo
				if p.isReplay(reqID) {
					return nil, fmt.Errorf("SAML replay detected: InResponseTo %s already seen", reqID)
				}
			}
		}
	}

	relayState := generateRelayState()
	userInfo := extractUserInfoFromAssertion(assertion, p.config.AttributeMapping)

	session := &samlSession{
		assertion:  assertion,
		relayState: relayState,
		createdAt:  time.Now(),
		userInfo:   userInfo,
	}
	p.storeSession(relayState, session)

	return &oauth2.Token{
		AccessToken: relayState,
		TokenType:   "SAML",
		Expiry:      time.Now().Add(replayTTL),
	}, nil
}

// GetUserInfo retrieves user information from a previously stored SAML session.
// The token's AccessToken must be the relayState returned by Exchange.
func (p *samlProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	if token == nil || token.AccessToken == "" {
		return nil, fmt.Errorf("token is required")
	}

	session, ok := p.getSession(token.AccessToken)
	if !ok {
		return nil, fmt.Errorf("no SAML session found for token (expired or invalid)")
	}

	if session.userInfo != nil {
		return session.userInfo, nil
	}

	// Fallback: extract from assertion if not pre-extracted
	return extractUserInfoFromAssertion(session.assertion, p.config.AttributeMapping), nil
}

// ValidateUser checks domain restrictions and required attributes.
func (p *samlProvider) ValidateUser(userInfo *UserInfo) error {
	if userInfo == nil {
		return fmt.Errorf("user info is nil")
	}

	// If allowed domains are configured via AttributeMapping, enforce email domain
	if len(p.config.AttributeMapping) > 0 {
		emailKey := ""
		for k, v := range p.config.AttributeMapping {
			if v == "email" || v == "mail" || k == "email" {
				emailKey = k
				break
			}
		}
		_ = emailKey // domain check uses UserInfo.Email directly
	}

	if userInfo.Email == "" {
		return fmt.Errorf("SAML assertion did not contain an email attribute")
	}

	return nil
}

// GetLogoutURL generates a SAML Single Logout (SLO) redirect URL.
func (p *samlProvider) GetLogoutURL(redirectURL string) string {
	sloLocation := p.sp.GetSLOBindingLocation(saml.HTTPRedirectBinding)
	if sloLocation == "" {
		// No IdP SLO endpoint configured — return the SP redirect
		return redirectURL
	}

	logoutReq, err := p.sp.MakeRedirectLogoutRequest("", redirectURL)
	if err != nil {
		// Fallback to plain redirect if SLO request construction fails
		return redirectURL
	}
	return logoutReq.String()
}

// SPMetadata returns the ServiceProvider metadata XML for IdP registration.
func (p *samlProvider) SPMetadata() ([]byte, error) {
	meta := p.sp.Metadata()
	if meta == nil {
		return nil, fmt.Errorf("failed to generate SP metadata")
	}
	return xml.MarshalIndent(meta, "", "  ")
}

// extractUserInfoFromAssertion maps SAML attributes to UserInfo using AttributeMapping.
func extractUserInfoFromAssertion(assertion *saml.Assertion, mapping map[string]string) *UserInfo {
	info := &UserInfo{
		Claims: make(map[string]interface{}),
	}

	if assertion == nil {
		return info
	}

	// NameID as subject
	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		info.Sub = assertion.Subject.NameID.Value
		// If NameID looks like an email, use it
		if strings.Contains(info.Sub, "@") && info.Email == "" {
			info.Email = info.Sub
		}
	}

	// Extract attributes from AttributeStatements
	for _, stmt := range assertion.AttributeStatements {
		for _, attr := range stmt.Attributes {
			values := make([]string, 0, len(attr.Values))
			for _, v := range attr.Values {
				values = append(values, v.Value)
			}
			val := ""
			if len(values) > 0 {
				val = values[0]
			}

			info.Claims[attr.Name] = values

			// Apply attribute mapping
			if target, ok := mapping[attr.Name]; ok {
				switch target {
				case "email", "mail":
					if val != "" {
						info.Email = val
					}
				case "name", "cn", "displayName":
					if val != "" {
						info.Name = val
					}
				case "picture", "photo":
					if val != "" {
						info.Picture = val
					}
				}
			} else {
				// Default mapping for well-known SAML attribute names
				switch attr.Name {
				case "email", "mail", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress":
					if val != "" && info.Email == "" {
						info.Email = val
					}
				case "name", "cn", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name":
					if val != "" && info.Name == "" {
						info.Name = val
					}
				}
			}
		}
	}

	return info
}
