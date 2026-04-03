package crypto_test

import (
	"bytes"
	"testing"

	"github.com/mimsewelt/1984/services/messaging/internal/crypto"
)

func TestGenerateDHKeyPair_PublicKeyIsNonZero(t *testing.T) {
	kp, err := crypto.GenerateDHKeyPair()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	zero := [32]byte{}
	if kp.Public == zero {
		t.Error("public key should not be all zeros")
	}
	if kp.Private == zero {
		t.Error("private key should not be all zeros")
	}
}

func TestGenerateDHKeyPair_EachCallProducesUniqueKeys(t *testing.T) {
	kp1, _ := crypto.GenerateDHKeyPair()
	kp2, _ := crypto.GenerateDHKeyPair()
	if kp1.Public == kp2.Public {
		t.Error("two key pairs should not share a public key")
	}
}

func TestGenerateOPKBatch_CorrectCount(t *testing.T) {
	batch, err := crypto.GenerateOPKBatch(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(batch) != 10 {
		t.Errorf("expected 10 OPKs, got %d", len(batch))
	}
}

func TestGenerateOPKBatch_AllUnique(t *testing.T) {
	batch, _ := crypto.GenerateOPKBatch(20)
	seen := make(map[[32]byte]bool)
	for _, kp := range batch {
		if seen[kp.Public] {
			t.Error("duplicate OPK public key in batch")
		}
		seen[kp.Public] = true
	}
}

func TestSignedPreKey_SignatureVerifies(t *testing.T) {
	ik, err := crypto.GenerateIdentityKeyPair()
	if err != nil {
		t.Fatalf("generate IK: %v", err)
	}
	spk, sig, err := crypto.GenerateSignedPreKey(ik, 1)
	if err != nil {
		t.Fatalf("generate SPK: %v", err)
	}
	ikSigningPub := []byte(ik.Signing)[32:]
	if !crypto.VerifySPKSignature(ikSigningPub, spk.Public[:], sig) {
		t.Error("SPK signature verification failed with correct key")
	}
}

func TestSignedPreKey_TamperedSPK_FailsVerification(t *testing.T) {
	ik, _ := crypto.GenerateIdentityKeyPair()
	spk, sig, _ := crypto.GenerateSignedPreKey(ik, 1)
	tampered := spk.Public
	tampered[0] ^= 0xFF
	ikPub := []byte(ik.Signing)[32:]
	if crypto.VerifySPKSignature(ikPub, tampered[:], sig) {
		t.Error("tampered SPK should fail signature verification")
	}
}

func TestSignedPreKey_WrongIK_FailsVerification(t *testing.T) {
	ik1, _ := crypto.GenerateIdentityKeyPair()
	ik2, _ := crypto.GenerateIdentityKeyPair()
	spk, sig, _ := crypto.GenerateSignedPreKey(ik1, 1)
	wrongIKPub := []byte(ik2.Signing)[32:]
	if crypto.VerifySPKSignature(wrongIKPub, spk.Public[:], sig) {
		t.Error("wrong IK should fail SPK verification")
	}
}

// buildBundle creates a PreKeyBundle with both DH and signing public keys separate.
func buildBundle(t *testing.T, recipientIK crypto.IdentityKeyPair, spk crypto.KeyPair, spkSig []byte, opk *crypto.KeyPair) crypto.PreKeyBundle {
	t.Helper()
	bundle := crypto.PreKeyBundle{
		UserID:         "recipient-user",
		IdentityKeyDH:  recipientIK.DH.Public[:],          // Curve25519 for DH
		IdentityKeySig: []byte(recipientIK.Signing)[32:],  // Ed25519 for signature verify
		SignedPreKey:   spk.Public[:],
		SPKSignature:   spkSig,
		SPKKeyID:       1,
	}
	if opk != nil {
		bundle.OneTimePreKey = opk.Public[:]
		bundle.OPKKeyID = 1
	}
	return bundle
}

func TestX3DH_SenderRecipientAgreeOnSameSecret(t *testing.T) {
	recipientIK, _ := crypto.GenerateIdentityKeyPair()
	spk, spkSig, _ := crypto.GenerateSignedPreKey(recipientIK, 1)
	opkBatch, _ := crypto.GenerateOPKBatch(1)
	opk := opkBatch[0]

	senderIK, _ := crypto.GenerateIdentityKeyPair()
	bundle := buildBundle(t, recipientIK, spk, spkSig, &opk)

	senderSecret, ephPub, err := crypto.X3DHSender(senderIK, bundle)
	if err != nil {
		t.Fatalf("X3DH sender failed: %v", err)
	}
	recipientSecret, err := crypto.X3DHRecipient(
		recipientIK, spk, &opk,
		senderIK.DH.Public[:], ephPub,
	)
	if err != nil {
		t.Fatalf("X3DH recipient failed: %v", err)
	}
	if !bytes.Equal(senderSecret, recipientSecret) {
		t.Errorf("X3DH secrets do not match:\n  sender:    %x\n  recipient: %x", senderSecret, recipientSecret)
	}
}

func TestX3DH_WithoutOPK_StillAgreesOnSecret(t *testing.T) {
	recipientIK, _ := crypto.GenerateIdentityKeyPair()
	spk, spkSig, _ := crypto.GenerateSignedPreKey(recipientIK, 1)
	senderIK, _ := crypto.GenerateIdentityKeyPair()
	bundle := buildBundle(t, recipientIK, spk, spkSig, nil)

	senderSecret, ephPub, err := crypto.X3DHSender(senderIK, bundle)
	if err != nil {
		t.Fatalf("sender failed: %v", err)
	}
	recipientSecret, err := crypto.X3DHRecipient(
		recipientIK, spk, nil,
		senderIK.DH.Public[:], ephPub,
	)
	if err != nil {
		t.Fatalf("recipient failed: %v", err)
	}
	if !bytes.Equal(senderSecret, recipientSecret) {
		t.Error("secrets mismatch without OPK")
	}
}

func TestX3DH_DifferentSessions_DifferentSecrets(t *testing.T) {
	recipientIK, _ := crypto.GenerateIdentityKeyPair()
	spk, spkSig, _ := crypto.GenerateSignedPreKey(recipientIK, 1)
	bundle := buildBundle(t, recipientIK, spk, spkSig, nil)

	senderIK1, _ := crypto.GenerateIdentityKeyPair()
	senderIK2, _ := crypto.GenerateIdentityKeyPair()

	secret1, _, _ := crypto.X3DHSender(senderIK1, bundle)
	secret2, _, _ := crypto.X3DHSender(senderIK2, bundle)

	if bytes.Equal(secret1, secret2) {
		t.Error("different senders should produce different shared secrets")
	}
}

func TestX3DH_TamperedSPKSignature_Rejected(t *testing.T) {
	recipientIK, _ := crypto.GenerateIdentityKeyPair()
	spk, spkSig, _ := crypto.GenerateSignedPreKey(recipientIK, 1)
	senderIK, _ := crypto.GenerateIdentityKeyPair()

	badSig := make([]byte, len(spkSig))
	copy(badSig, spkSig)
	badSig[0] ^= 0xFF

	bundle := buildBundle(t, recipientIK, spk, badSig, nil)
	_, _, err := crypto.X3DHSender(senderIK, bundle)
	if err == nil {
		t.Error("expected error for tampered SPK signature, got nil")
	}
}

func TestX3DH_SecretIs32Bytes(t *testing.T) {
	recipientIK, _ := crypto.GenerateIdentityKeyPair()
	spk, spkSig, _ := crypto.GenerateSignedPreKey(recipientIK, 1)
	senderIK, _ := crypto.GenerateIdentityKeyPair()
	bundle := buildBundle(t, recipientIK, spk, spkSig, nil)

	secret, ephPub, err := crypto.X3DHSender(senderIK, bundle)
	if err != nil {
		t.Fatalf("X3DH sender failed: %v", err)
	}
	if len(secret) != 32 {
		t.Errorf("expected 32-byte shared secret, got %d", len(secret))
	}
	if len(ephPub) != 32 {
		t.Errorf("expected 32-byte ephemeral public key, got %d", len(ephPub))
	}
}

func TestEncodeDecodeKey_RoundTrip(t *testing.T) {
	kp, _ := crypto.GenerateDHKeyPair()
	encoded := crypto.EncodeKey(kp.Public[:])
	decoded, err := crypto.DecodeKey(encoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !bytes.Equal(decoded, kp.Public[:]) {
		t.Error("key encode/decode round-trip failed")
	}
}

func TestDecodeKey_InvalidBase64_ReturnsError(t *testing.T) {
	_, err := crypto.DecodeKey("not!!valid!!base64@@")
	if err == nil {
		t.Error("expected error for invalid base64, got nil")
	}
}
