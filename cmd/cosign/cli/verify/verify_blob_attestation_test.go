// Copyright 2022 the Sigstore Authors.
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

package verify

import (
	"bytes"
	"context"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	ctypes "github.com/sigstore/cosign/v2/pkg/types"
	"github.com/sigstore/cosign/v2/test"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/dsse"
	signatureoptions "github.com/sigstore/sigstore/pkg/signature/options"
)

const pubkey = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAESF79b1ToAtoakhBOHEU5UjnEiihV
gZPFIp557+TOoDxf14FODWc+sIPETk0OgCplAk60doVXbCv33IU4rXZHrg==
-----END PUBLIC KEY-----
`

const (
	blobContents                         = "some-payload"
	anotherBlobContents                  = "another-blob"
	blobSLSAProvenanceSignature          = "eyJwYXlsb2FkVHlwZSI6ImFwcGxpY2F0aW9uL3ZuZC5pbi10b3RvK2pzb24iLCJwYXlsb2FkIjoiZXlKZmRIbHdaU0k2SW1oMGRIQnpPaTh2YVc0dGRHOTBieTVwYnk5VGRHRjBaVzFsYm5RdmRqQXVNU0lzSW5CeVpXUnBZMkYwWlZSNWNHVWlPaUpvZEhSd2N6b3ZMM05zYzJFdVpHVjJMM0J5YjNabGJtRnVZMlV2ZGpBdU1pSXNJbk4xWW1wbFkzUWlPbHQ3SW01aGJXVWlPaUppYkc5aUlpd2laR2xuWlhOMElqcDdJbk5vWVRJMU5pSTZJalkxT0RjNE1XTmtOR1ZrT1dKallUWXdaR0ZqWkRBNVpqZGlZamt4TkdKaU5URTFNREpsT0dJMVpEWXhPV1kxTjJZek9XRXhaRFkxTWpVNU5tTmpNalFpZlgxZExDSndjbVZrYVdOaGRHVWlPbnNpWW5WcGJHUmxjaUk2ZXlKcFpDSTZJaklpZlN3aVluVnBiR1JVZVhCbElqb2llQ0lzSW1sdWRtOWpZWFJwYjI0aU9uc2lZMjl1Wm1sblUyOTFjbU5sSWpwN2ZYMTlmUT09Iiwic2lnbmF0dXJlcyI6W3sia2V5aWQiOiIiLCJzaWciOiJNRVVDSUE4S2pacWtydDkwZnpCb2pTd3d0ajNCcWI0MUU2cnV4UWs5N1RMbnB6ZFlBaUVBek9Bak9Uenl2VEhxYnBGREFuNnpocmc2RVp2N2t4SzVmYVJvVkdZTWgyYz0ifV19"
	dssePredicateEmptySubject            = "eyJwYXlsb2FkVHlwZSI6ImFwcGxpY2F0aW9uL3ZuZC5pbi10b3RvK2pzb24iLCJwYXlsb2FkIjoiZXlKZmRIbHdaU0k2SW1oMGRIQnpPaTh2YVc0dGRHOTBieTVwYnk5VGRHRjBaVzFsYm5RdmRqQXVNU0lzSW5CeVpXUnBZMkYwWlZSNWNHVWlPaUpvZEhSd2N6b3ZMM05zYzJFdVpHVjJMM0J5YjNabGJtRnVZMlV2ZGpBdU1pSXNJbk4xWW1wbFkzUWlPbHRkTENKd2NtVmthV05oZEdVaU9uc2lZblZwYkdSbGNpSTZleUpwWkNJNklqSWlmU3dpWW5WcGJHUlVlWEJsSWpvaWVDSXNJbWx1ZG05allYUnBiMjRpT25zaVkyOXVabWxuVTI5MWNtTmxJanA3ZlgxOWZRPT0iLCJzaWduYXR1cmVzIjpbeyJrZXlpZCI6IiIsInNpZyI6Ik1FWUNJUUNrTEV2NkhZZ0svZDdUK0N3NTdXbkZGaHFUTC9WalAyVDA5Q2t1dk1nbDRnSWhBT1hBM0lhWWg1M1FscVk1eVU4cWZxRXJma2tGajlEakZnaWovUTQ2NnJSViJ9XX0="
	dssePredicateMissingSha256           = "eyJwYXlsb2FkVHlwZSI6ImFwcGxpY2F0aW9uL3ZuZC5pbi10b3RvK2pzb24iLCJwYXlsb2FkIjoiZXlKZmRIbHdaU0k2SW1oMGRIQnpPaTh2YVc0dGRHOTBieTVwYnk5VGRHRjBaVzFsYm5RdmRqQXVNU0lzSW5CeVpXUnBZMkYwWlZSNWNHVWlPaUpvZEhSd2N6b3ZMM05zYzJFdVpHVjJMM0J5YjNabGJtRnVZMlV2ZGpBdU1pSXNJbk4xWW1wbFkzUWlPbHQ3SW01aGJXVWlPaUppYkc5aUlpd2laR2xuWlhOMElqcDdmWDFkTENKd2NtVmthV05oZEdVaU9uc2lZblZwYkdSbGNpSTZleUpwWkNJNklqSWlmU3dpWW5WcGJHUlVlWEJsSWpvaWVDSXNJbWx1ZG05allYUnBiMjRpT25zaVkyOXVabWxuVTI5MWNtTmxJanA3ZlgxOWZRPT0iLCJzaWduYXR1cmVzIjpbeyJrZXlpZCI6IiIsInNpZyI6Ik1FVUNJQysvM2M4RFo1TGFZTEx6SFZGejE3ZmxHUENlZXVNZ2tIKy8wa2s1cFFLUEFpRUFqTStyYnBBRlJybDdpV0I2Vm9BYVZPZ3U3NjRRM0JKdHI1bHk4VEFHczNrPSJ9XX0="
	dssePredicateMultipleSubjects        = "eyJwYXlsb2FkVHlwZSI6ImFwcGxpY2F0aW9uL3ZuZC5pbi10b3RvK2pzb24iLCJwYXlsb2FkIjoiZXlKZmRIbHdaU0k2SW1oMGRIQnpPaTh2YVc0dGRHOTBieTVwYnk5VGRHRjBaVzFsYm5RdmRqQXVNU0lzSW5CeVpXUnBZMkYwWlZSNWNHVWlPaUpvZEhSd2N6b3ZMM05zYzJFdVpHVjJMM0J5YjNabGJtRnVZMlV2ZGpBdU1pSXNJbk4xWW1wbFkzUWlPbHQ3SW01aGJXVWlPaUppYkc5aUlpd2laR2xuWlhOMElqcDdJbk5vWVRJMU5pSTZJalkxT0RjNE1XTmtOR1ZrT1dKallUWXdaR0ZqWkRBNVpqZGlZamt4TkdKaU5URTFNREpsT0dJMVpEWXhPV1kxTjJZek9XRXhaRFkxTWpVNU5tTmpNalFpZlgwc2V5SnVZVzFsSWpvaWIzUm9aWElpTENKa2FXZGxjM1FpT25zaWMyaGhNalUySWpvaU1HUmhOVFU1WXpKbU1USTNNak13WVRGbVlXSmpabUppTWpCa05XUmlPR1JpWVRjMk5Ua3lNMk0yWldaak5tWTBPRE14TmpVeE1UbGpOR015WXpWa05DSjlmVjBzSW5CeVpXUnBZMkYwWlNJNmV5SmlkV2xzWkdWeUlqcDdJbWxrSWpvaU1pSjlMQ0ppZFdsc1pGUjVjR1VpT2lKNElpd2lhVzUyYjJOaGRHbHZiaUk2ZXlKamIyNW1hV2RUYjNWeVkyVWlPbnQ5ZlgxOSIsInNpZ25hdHVyZXMiOlt7ImtleWlkIjoiIiwic2lnIjoiTUVZQ0lRQ20yR2FwNzRzbDkyRC80V2FoWHZiVHFrNFVCaHZsb3oreDZSZm1NQXUyaWdJaEFNcXRFV29DalpGdkpmZWJxRDJFank3aTlHaGc0a0V0WE51bVdLbVBtdEphIn1dfQ=="
	dssePredicateMultipleSubjectsInvalid = "eyJwYXlsb2FkVHlwZSI6ImFwcGxpY2F0aW9uL3ZuZC5pbi10b3RvK2pzb24iLCJwYXlsb2FkIjoiZXlKZmRIbHdaU0k2SW1oMGRIQnpPaTh2YVc0dGRHOTBieTVwYnk5VGRHRjBaVzFsYm5RdmRqQXVNU0lzSW5CeVpXUnBZMkYwWlZSNWNHVWlPaUpvZEhSd2N6b3ZMM05zYzJFdVpHVjJMM0J5YjNabGJtRnVZMlV2ZGpBdU1pSXNJbk4xWW1wbFkzUWlPbHQ3SW01aGJXVWlPaUppYkc5aUlpd2laR2xuWlhOMElqcDdJbk5vWVRJMU5pSTZJbUUyT0RJelpqbGpOekEyTWpCalltWmpOVGt4T0dJMVpUWmtOR0ZoTVRjMFlUaGhNakJrTlRaa1lUVm1NVEEyWWpZMU5qSTNOR013TldRMlptVXhZVGNpZlgwc2V5SnVZVzFsSWpvaWIzUm9aWElpTENKa2FXZGxjM1FpT25zaWMyaGhNalUySWpvaU1HUmhOVFU1WXpKbU1USTNNak13WVRGbVlXSmpabUppTWpCa05XUmlPR1JpWVRjMk5Ua3lNMk0yWldaak5tWTBPRE14TmpVeE1UbGpOR015WXpWa05DSjlmVjBzSW5CeVpXUnBZMkYwWlNJNmV5SmlkV2xzWkdWeUlqcDdJbWxrSWpvaU1pSjlMQ0ppZFdsc1pGUjVjR1VpT2lKNElpd2lhVzUyYjJOaGRHbHZiaUk2ZXlKamIyNW1hV2RUYjNWeVkyVWlPbnQ5ZlgxOSIsInNpZ25hdHVyZXMiOlt7ImtleWlkIjoiIiwic2lnIjoiTUVVQ0lRRGhZbCtWUlBtcWFJc2xxdS9yWGRVbnc2VmpQcXR4RG84bHdqc3p1cWl6MmdJZ0NNRVVlcUZ5RkFZejcyM2IvSTI2L0p3K0U3YkFLMExqeElsUExvTGxPczQ9In1dfQ=="
)

func TestVerifyBlobAttestation(t *testing.T) {
	ctx := context.Background()
	td := t.TempDir()
	defer os.RemoveAll(td)

	blobPath := writeBlobFile(t, td, blobContents, "blob")
	anotherBlobPath := writeBlobFile(t, td, anotherBlobContents, "other-blob")
	keyRef := writeBlobFile(t, td, pubkey, "cosign.pub")

	tests := []struct {
		description   string
		blobPath      string
		signature     string
		predicateType string
		shouldErr     bool
	}{
		{
			description:   "verify a slsaprovenance predicate",
			predicateType: "slsaprovenance",
			blobPath:      blobPath,
			signature:     blobSLSAProvenanceSignature,
		}, {
			description:   "fail with incorrect predicate",
			signature:     blobSLSAProvenanceSignature,
			blobPath:      blobPath,
			predicateType: "custom",
			shouldErr:     true,
		}, {
			description: "fail with incorrect blob",
			signature:   blobSLSAProvenanceSignature,
			blobPath:    anotherBlobPath,
			shouldErr:   true,
		}, {
			description: "dsse envelope predicate has no subject",
			signature:   dssePredicateEmptySubject,
			blobPath:    blobPath,
			shouldErr:   true,
		}, {
			description: "dsse envelope predicate missing sha256 digest",
			signature:   dssePredicateMissingSha256,
			blobPath:    blobPath,
			shouldErr:   true,
		}, {
			description:   "dsse envelope has multiple subjects, one is valid",
			predicateType: "slsaprovenance",
			signature:     dssePredicateMultipleSubjects,
			blobPath:      blobPath,
		}, {
			description:   "dsse envelope has multiple subjects, one is valid, but we are looking for different predicatetype",
			predicateType: "notreallyslsaprovenance",
			signature:     dssePredicateMultipleSubjects,
			blobPath:      blobPath,
			shouldErr:     true,
		}, {
			description:   "dsse envelope has multiple subjects, none has correct sha256 digest",
			predicateType: "slsaprovenance",
			signature:     dssePredicateMultipleSubjectsInvalid,
			blobPath:      blobPath,
			shouldErr:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			decodedSig, err := base64.StdEncoding.DecodeString(test.signature)
			if err != nil {
				t.Fatal(err)
			}
			sigRef := writeBlobFile(t, td, string(decodedSig), "signature")

			cmd := VerifyBlobAttestationCommand{
				KeyOpts:       options.KeyOpts{KeyRef: keyRef},
				SignaturePath: sigRef,
				IgnoreTlog:    true,
				CheckClaims:   true,
				PredicateType: test.predicateType,
			}
			err = cmd.Exec(ctx, test.blobPath)

			if (err != nil) != test.shouldErr {
				t.Fatalf("verifyBlobAttestation()= %s, expected shouldErr=%t ", err, test.shouldErr)
			}
		})
	}
}

func TestVerifyBlobAttestationNoCheckClaims(t *testing.T) {
	ctx := context.Background()
	td := t.TempDir()
	defer os.RemoveAll(td)

	blobPath := writeBlobFile(t, td, blobContents, "blob")
	anotherBlobPath := writeBlobFile(t, td, anotherBlobContents, "other-blob")
	keyRef := writeBlobFile(t, td, pubkey, "cosign.pub")

	tests := []struct {
		description string
		blobPath    string
		signature   string
	}{
		{
			description: "verify a predicate",
			blobPath:    blobPath,
			signature:   blobSLSAProvenanceSignature,
		}, {
			description: "verify a predicate no path",
			signature:   blobSLSAProvenanceSignature,
		}, {
			description: "verify a predicate with another blob path",
			signature:   blobSLSAProvenanceSignature,
			// This works because we're not checking the claims. It doesn't matter what we put in here - it should pass so long as the DSSE signagure can be verified.
			blobPath: anotherBlobPath,
		}, {
			description: "verify a predicate with /dev/null",
			signature:   blobSLSAProvenanceSignature,
			blobPath:    "/dev/null",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			decodedSig, err := base64.StdEncoding.DecodeString(test.signature)
			if err != nil {
				t.Fatal(err)
			}
			sigRef := writeBlobFile(t, td, string(decodedSig), "signature")

			cmd := VerifyBlobAttestationCommand{
				KeyOpts:       options.KeyOpts{KeyRef: keyRef},
				SignaturePath: sigRef,
				IgnoreTlog:    true,
				CheckClaims:   false,
				PredicateType: "slsaprovenance",
			}
			if err := cmd.Exec(ctx, test.blobPath); err != nil {
				t.Fatalf("verifyBlobAttestation()= %v", err)
			}
		})
	}
}

func TestVerifyBlobAttestationOfflineChain(t *testing.T) {
	ctx := context.Background()
	td := t.TempDir()

	rootCert, rootPriv, err := test.GenerateRootCa()
	if err != nil {
		t.Fatal(err)
	}

	subCert, subPriv, err := test.GenerateSubordinateCa(rootCert, rootPriv)
	if err != nil {
		t.Fatal(err)
	}

	leafCert, leafPriv, err := test.GenerateLeafCert("leaf-subject", "leaf-odic-issuer", subCert, subPriv)
	if err != nil {
		t.Fatal(err)
	}
	leafPEM, err := cryptoutils.MarshalCertificateToPEM(leafCert)
	if err != nil {
		t.Fatal(err)
	}
	leafPath := writeBlobFile(t, td, string(leafPEM), "leafcert.pem")

	signer, err := signature.LoadECDSASignerVerifier(leafPriv, crypto.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	var makeSignature = func(blob []byte) string {
		stmt := `{"_type":"https://in-toto.io/Statement/v0.1","predicateType":"customFoo","subject":[{"name":"subject","digest":{"sha256":"deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"}}],"predicate":{}}`
		wrapped := dsse.WrapSigner(signer, ctypes.IntotoPayloadType)
		sig, err := wrapped.SignMessage(bytes.NewReader([]byte(stmt)), signatureoptions.WithContext(context.Background()))
		if err != nil {
			t.Fatal(err)
		}
		return string(sig)
	}

	blobBytes := []byte("foo")
	blobSignature := makeSignature(blobBytes)
	blobPath := writeBlobFile(t, td, string(blobBytes), "blob.txt")
	sigPath := writeBlobFile(t, td, blobSignature, "signature.txt")

	tts := []struct {
		name       string
		chainCerts []*x509.Certificate
		shouldErr  bool
	}{
		{
			name:       "complete chain works",
			chainCerts: []*x509.Certificate{subCert, rootCert},
			shouldErr:  false,
		},
		{
			name:       "no intermediate fails",
			chainCerts: []*x509.Certificate{rootCert},
			shouldErr:  true,
		},
		{
			// NOTE: This case actually passes with current usage!
			// We assume the last entry in the chain _is_ a root, even
			// if it's not self-signed.  So, while we'd probably
			// prefer this to fail, it doesn't and we probably have
			// to resolve elsewhere as noted in https://github.com/sigstore/cosign/issues/3462#issuecomment-1893129844
			name:       "no root fails",
			chainCerts: []*x509.Certificate{subCert},
			shouldErr:  false,
		},
	}
	for tn, tt := range tts {
		t.Run(tt.name, func(t *testing.T) {
			tt := tt

			chainPath, err := writeChain(t, td, fmt.Sprintf("chain-%d.pem", tn), tt.chainCerts)
			if err != nil {
				t.Fatal(err)
			}

			cmd := VerifyBlobAttestationCommand{
				CertVerifyOptions: options.CertVerifyOptions{
					CertIdentityRegexp:   ".*",
					CertOidcIssuerRegexp: ".*",
				},
				CertRef:       leafPath,
				CertChain:     chainPath,
				SignaturePath: sigPath,
				IgnoreSCT:     true,
				IgnoreTlog:    true,
				CheckClaims:   false,
				PredicateType: "customFoo",
			}
			err = cmd.Exec(ctx, blobPath)

			if (err != nil) != tt.shouldErr {
				t.Fatalf("verifyBlobAttestation()= %s, expected shouldErr=%t ", err, tt.shouldErr)
			}
		})
	}
}
