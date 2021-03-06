package proof

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ed25519"
)

type provableTestData struct {
	A     string `json:"a"`
	B     string `json:"b,omitempty"`
	Proof *Proof `json:"proof,omitempty"`
}

func (t *provableTestData) GetProof() *Proof {
	return t.Proof
}

func (t *provableTestData) SetProof(p *Proof) {
	t.Proof = p
}

func TestSignatureSuiteFactory_GetSuiteForProof(t *testing.T) {
	const (
		vMethod       = "verificationMethod"
		creator       = "creator"
		proofTemplate = `{
			"created": "2020-06-05T01:12:15Z",
			"%s": "key-1",
			"nonce": "015b5f58-ba8d-4da5-b278-b4a095e09e9c",
			"signatureValue": "5Hj1yvfw9LMMd656K2gKxhUfFGNrVUqxyhFGw72ZkYwANtrz3PibuFYQmCSLeAhHfRYmbbHbiyUAmwKxdZtK1YfP",
			"type": "%s"
		}`
	)
	inputs := []struct {
		sigType       SignatureType
		expectedSuite SignatureSuite
		keyRefType    string
		err           bool
	}{
		{sigType: JCSEdSignatureType, expectedSuite: jcsEd25519SignatureSuite, keyRefType: vMethod},
		{sigType: EcdsaSecp256k1SignatureType, expectedSuite: secp256K1SignatureSuite, keyRefType: creator},
		{sigType: WorkEdSignatureType, expectedSuite: workSignatureSuiteV1, keyRefType: creator},
		{sigType: Ed25519SignatureType, expectedSuite: ed25519SignatureSuiteV1, keyRefType: creator},
		{sigType: WorkEdSignatureType, expectedSuite: workSignatureSuiteV2, keyRefType: vMethod},
		{sigType: Ed25519SignatureType, expectedSuite: ed25519SignatureSuiteV2, keyRefType: vMethod},
		{sigType: JCSEdSignatureType, expectedSuite: jcsEd25519SignatureSuite, keyRefType: creator, err: true},
		{sigType: EcdsaSecp256k1SignatureType, expectedSuite: secp256K1SignatureSuite, keyRefType: vMethod, err: true},
	}
	for _, input := range inputs {
		name := fmt.Sprintf("%s-%s", input.sigType, input.keyRefType)
		t.Run(name, func(t *testing.T) {
			proofJSON := fmt.Sprintf(proofTemplate, input.keyRefType, input.sigType)
			var p Proof
			assert.NoError(t, json.Unmarshal([]byte(proofJSON), &p))
			suite, err := SignatureSuites().GetSuiteForProof(&p)
			if err != nil {
				assert.True(t, strings.Contains(err.Error(), "unsupported signature type"))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, input.expectedSuite, suite)
			}
		})
	}
}

var (
	seed    = []byte("12345678901234567890123456789012")
	privKey = ed25519.NewKeyFromSeed(seed)
	pubKey  = privKey.Public().(ed25519.PublicKey)
)

func TestSignAndVerify(t *testing.T) {
	js := `{
		"a": "hello",
		"b": "world"
	}`

	privKey := ed25519.NewKeyFromSeed(seed)
	pubKey := privKey.Public().(ed25519.PublicKey)

	signer, err := NewEd25519Signer(privKey, "key-1")
	assert.NoError(t, err)

	verifier := &Ed25519Verifier{PubKey: pubKey}

	suites := map[string]SignatureSuite{
		"JCS":       jcsEd25519SignatureSuite,
		"WorkV1":    workSignatureSuiteV1,
		"WorkV2":    workSignatureSuiteV2,
		"Ed25519V1": ed25519SignatureSuiteV1,
		"Ed25519V2": ed25519SignatureSuiteV2,
	}
	for name, suite := range suites {
		t.Run(name, func(t *testing.T) {
			var provable provableTestData
			assert.NoError(t, json.Unmarshal([]byte(js), &provable))
			assert.NoError(t, suite.Sign(&provable, signer))
			assert.NoError(t, suite.Verify(&provable, verifier))
		})
	}

	t.Run("Secp256K1_Error", func(t *testing.T) {
		var provable provableTestData
		assert.NoError(t, json.Unmarshal([]byte(js), &provable))
		assert.EqualError(t, secp256K1SignatureSuite.Sign(&provable, signer), "incorrect key type")
	})
}

func TestVerify(t *testing.T) {
	privKey := ed25519.NewKeyFromSeed(seed)
	pubKey := privKey.Public().(ed25519.PublicKey)
	verifier := &Ed25519Verifier{PubKey: pubKey}

	inputs := map[string]string{
		"JCS": `{
			"a": "hello",
			"b": "world",
			"proof": {
				"created": "2020-06-05T01:12:15Z",
				"verificationMethod": "key-1",
				"nonce": "015b5f58-ba8d-4da5-b278-b4a095e09e9c",
				"signatureValue": "5Hj1yvfw9LMMd656K2gKxhUfFGNrVUqxyhFGw72ZkYwANtrz3PibuFYQmCSLeAhHfRYmbbHbiyUAmwKxdZtK1YfP",
				"type": "JcsEd25519Signature2020"
			}
		}`,
		"WorkV1": `{
			"a": "hello",
			"b": "world",
			"proof": {
				"created": "2020-06-05T00:12:14Z",
				"creator": "key-1",
				"nonce": "e11cc825-2de1-4719-a6d8-97ae4b10ab6b",
				"signatureValue": "3zJDnaLEsT2c27AYh1GjdQGPte2ezS5vFxDC2pxgSsXg41ofRwfUK7iPdMxeak3kcruELckU72nNw73MEfcbsiVN",
				"type": "WorkEd25519Signature2020"
			}
		}`,
		"WorkV2": `{
			"a": "hello",
			"b": "world",
			"proof": {
				"created": "2020-06-05T00:13:43Z",
				"verificationMethod": "key-1",
				"nonce": "3acd37af-8e01-4dbe-aa50-32337527c702",
				"signatureValue": "3HY91yqeN6ST8hDq8TyYpfHyNUUtjLjwZLPoe2tH6pZgpFiqZnuycEhiqxz7yJ46mKYuVPEbUTxK9AxBHcWwiCpq",
				"type": "WorkEd25519Signature2020"
			}
		}`,
	}
	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			var provable provableTestData
			assert.NoError(t, json.Unmarshal([]byte(input), &provable))

			suite, err := SignatureSuites().GetSuiteForProof(provable.GetProof())
			assert.NoError(t, err)
			assert.NoError(t, suite.Verify(&provable, verifier))

			provable.GetProof().SignatureValue = "bogus"
			assert.EqualError(t, suite.Verify(&provable, verifier), "signature verification failed")
		})
	}
}

type workEd25519Signer struct {
	Ed25519Signer
}

func (s *workEd25519Signer) Type() KeyType                      {
	return WorkEdKeyType
}

type workEd25519Verifier struct {
	Ed25519Verifier
}

func (v *workEd25519Verifier) Type() KeyType {
	return WorkEdKeyType
}

func TestCreateAndVerifyJCSEd25519Proof(t *testing.T) {
	testSigner, err := NewEd25519Signer(privKey, "key-ref")
	assert.NoError(t, err)
	testVerifier := &Ed25519Verifier{PubKey: pubKey}

	t.Run("Happy path sign and verify", func(t *testing.T) {
		testData := GenericProvable{JSONData: "testData"}

		suite, err := SignatureSuites().GetSuite(JCSEdSignatureType, V2)
		assert.NoError(t, err)
		// Create and set proof
		err = suite.Sign(&testData, testSigner)
		assert.NoError(t, err)

		err = suite.Verify(&testData, testVerifier)
		assert.NoError(t, err)
	})

	t.Run("Bad pub key - can't verify", func(t *testing.T) {
		testData := GenericProvable{JSONData: "testData"}

		// Create and set proof
		suite, err := SignatureSuites().GetSuite(JCSEdSignatureType, V2)
		assert.NoError(t, err)
		// Create and set proof
		err = suite.Sign(&testData, testSigner)
		assert.NoError(t, err)

		badPubKey := make([]byte, ed25519.PublicKeySize)
		testVerifier := &Ed25519Verifier{PubKey: badPubKey}
		err = suite.Verify(&testData, testVerifier)
		assert.Error(t, err)
	})

	t.Run("Non Ed25519 Signer", func(t *testing.T) {
		testData := GenericProvable{JSONData: "testData"}

		badSigner := &workEd25519Signer{Ed25519Signer: Ed25519Signer{KeyID: "key-ref", PrivateKey: privKey}}
		suite, err := SignatureSuites().GetSuite(JCSEdSignatureType, V2)
		assert.NoError(t, err)

		err = suite.Sign(&testData, badSigner)
		assert.Error(t, err)
	})

	t.Run("Non Ed25519 Verifier", func(t *testing.T) {
		testData := GenericProvable{JSONData: "testData"}

		suite, err := SignatureSuites().GetSuite(JCSEdSignatureType, V2)
		assert.NoError(t, err)

		err = suite.Sign(&testData, testSigner)
		assert.NoError(t, err)

		badVerifier := &workEd25519Verifier{Ed25519Verifier: Ed25519Verifier{PubKey: pubKey}}
		err = suite.Verify(&testData, badVerifier)
		assert.NoError(t, err)
	})
}
