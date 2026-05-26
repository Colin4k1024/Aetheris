# Evidence Signing Release Drill

- Timestamp: 20260525-131432
- Overall: PASS

## Suites

- internal/app/api (PASS)
- internal/api/http (PASS)
- cmd/cli (PASS)
- pkg/proof (PASS)

## internal/app/api output

```text
=== RUN   TestEvidenceSigningConfigFromConfig
--- PASS: TestEvidenceSigningConfigFromConfig (0.00s)
=== RUN   TestEvidenceSigningConfigFromConfig_InvalidKey
--- PASS: TestEvidenceSigningConfigFromConfig_InvalidKey (0.00s)
PASS
ok  	github.com/Colin4k1024/Aetheris/v2/internal/app/api	1.202s
```

## internal/api/http output

```text
=== RUN   TestBuildForensicsPackage_ProofCompatible
--- PASS: TestBuildForensicsPackage_ProofCompatible (0.00s)
=== RUN   TestBuildForensicsPackage_SignedProof
--- PASS: TestBuildForensicsPackage_SignedProof (0.00s)
=== RUN   TestBuildForensicsPackage_InvalidSigningKey
--- PASS: TestBuildForensicsPackage_InvalidSigningKey (0.00s)
PASS
ok  	github.com/Colin4k1024/Aetheris/v2/internal/api/http	0.653s
```

## cmd/cli output

```text
=== RUN   TestVerifyEvidenceZip_Success
--- PASS: TestVerifyEvidenceZip_Success (0.00s)
=== RUN   TestVerifyEvidenceZip_Tampered
--- PASS: TestVerifyEvidenceZip_Tampered (0.00s)
=== RUN   TestVerifyEvidenceZip_SignedWithPublicKey
--- PASS: TestVerifyEvidenceZip_SignedWithPublicKey (0.00s)
=== RUN   TestParseEvidenceVerifyArgs_PublicKey
--- PASS: TestParseEvidenceVerifyArgs_PublicKey (0.00s)
PASS
ok  	github.com/Colin4k1024/Aetheris/v2/cmd/cli	0.487s
```

## pkg/proof output

```text
=== RUN   TestEndToEnd_ExportAndVerify
--- PASS: TestEndToEnd_ExportAndVerify (0.00s)
=== RUN   TestEndToEnd_TamperDetection
--- PASS: TestEndToEnd_TamperDetection (0.00s)
PASS
ok  	github.com/Colin4k1024/Aetheris/v2/pkg/proof	0.457s
```
