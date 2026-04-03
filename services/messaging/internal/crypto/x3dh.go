package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

const (
	OPKBatchSize    = 100
	SPKRotateDays   = 7
	OPKLowThreshold = 10
)

type KeyPair struct {
	Private [32]byte
	Public  [32]byte
}

type IdentityKeyPair struct {
	DH      KeyPair            // Curve25519 — used in X3DH key exchange
	Signing ed25519.PrivateKey // Ed25519 — used to sign SPK
}

// PreKeyBundle is published per user on the server.
// IdentityKeyDH  = Curve25519 public key (for DH exchange)
// IdentityKeySig = Ed25519 public key (for SPK signature verification)
type PreKeyBundle struct {
	UserID         string    `json:"user_id"`
	IdentityKeyDH  []byte    `json:"ik_dh"`   // Curve25519 public key
	IdentityKeySig []byte    `json:"ik_sig"`  // Ed25519 public key
	SignedPreKey   []byte    `json:"spk"`
	SPKSignature   []byte    `json:"spk_sig"`
	SPKKeyID       uint32    `json:"spk_id"`
	OneTimePreKey  []byte    `json:"opk"`
	OPKKeyID       uint32    `json:"opk_id"`
	PublishedAt    time.Time `json:"published_at"`
}

func GenerateDHKeyPair() (KeyPair, error) {
	var priv [32]byte
	if _, err := rand.Read(priv[:]); err != nil {
		return KeyPair{}, err
	}
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	pub, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return KeyPair{}, err
	}
	var kp KeyPair
	copy(kp.Private[:], priv[:])
	copy(kp.Public[:], pub)
	return kp, nil
}

func GenerateIdentityKeyPair() (IdentityKeyPair, error) {
	dh, err := GenerateDHKeyPair()
	if err != nil {
		return IdentityKeyPair{}, err
	}
	_, sigPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return IdentityKeyPair{}, err
	}
	return IdentityKeyPair{DH: dh, Signing: sigPriv}, nil
}

func GenerateSignedPreKey(ik IdentityKeyPair, keyID uint32) (KeyPair, []byte, error) {
	spk, err := GenerateDHKeyPair()
	if err != nil {
		return KeyPair{}, nil, err
	}
	sig := ed25519.Sign(ik.Signing, spk.Public[:])
	return spk, sig, nil
}

func VerifySPKSignature(ikSigningPub []byte, spkPublic, sig []byte) bool {
	return ed25519.Verify(ed25519.PublicKey(ikSigningPub), spkPublic, sig)
}

func GenerateOPKBatch(n int) ([]KeyPair, error) {
	keys := make([]KeyPair, n)
	for i := range keys {
		kp, err := GenerateDHKeyPair()
		if err != nil {
			return nil, err
		}
		keys[i] = kp
	}
	return keys, nil
}

// X3DHSender performs X3DH as the initiating party.
// DH1 = DH(IK_A_dh, SPK_B)
// DH2 = DH(EK_A,    IK_B_dh)
// DH3 = DH(EK_A,    SPK_B)
// DH4 = DH(EK_A,    OPK_B)  [optional]
func X3DHSender(
	senderIK IdentityKeyPair,
	bundle PreKeyBundle,
) (sharedSecret []byte, ephemeralPub []byte, err error) {
	// Verify SPK was signed by recipient's Ed25519 key.
	if !VerifySPKSignature(bundle.IdentityKeySig, bundle.SignedPreKey, bundle.SPKSignature) {
		return nil, nil, errors.New("x3dh: SPK signature invalid")
	}

	ek, err := GenerateDHKeyPair()
	if err != nil {
		return nil, nil, err
	}

	dh1, err := dh(senderIK.DH.Private[:], bundle.SignedPreKey)   // IK_A × SPK_B
	if err != nil { return nil, nil, err }
	dh2, err := dh(ek.Private[:], bundle.IdentityKeyDH)           // EK_A × IK_B_dh
	if err != nil { return nil, nil, err }
	dh3, err := dh(ek.Private[:], bundle.SignedPreKey)             // EK_A × SPK_B
	if err != nil { return nil, nil, err }

	input := append(dh1, dh2...)
	input = append(input, dh3...)

	if len(bundle.OneTimePreKey) == 32 {
		dh4, err := dh(ek.Private[:], bundle.OneTimePreKey)        // EK_A × OPK_B
		if err != nil { return nil, nil, err }
		input = append(input, dh4...)
	}

	return kdf(input, "InstagramClone_X3DH_v1"), ek.Public[:], nil
}

// X3DHRecipient performs X3DH as the receiving party.
// DH1 = DH(SPK_B,  IK_A_dh)
// DH2 = DH(IK_B_dh, EK_A)
// DH3 = DH(SPK_B,  EK_A)
// DH4 = DH(OPK_B,  EK_A)   [optional]
func X3DHRecipient(
	recipientIK IdentityKeyPair,
	recipientSPK KeyPair,
	recipientOPK *KeyPair,
	senderIKDHPublic []byte, // sender's Curve25519 DH public key
	ephemeralPublic []byte,
) (sharedSecret []byte, err error) {
	dh1, err := dh(recipientSPK.Private[:], senderIKDHPublic)    // SPK_B × IK_A_dh
	if err != nil { return nil, err }
	dh2, err := dh(recipientIK.DH.Private[:], ephemeralPublic)   // IK_B_dh × EK_A
	if err != nil { return nil, err }
	dh3, err := dh(recipientSPK.Private[:], ephemeralPublic)      // SPK_B × EK_A
	if err != nil { return nil, err }

	input := append(dh1, dh2...)
	input = append(input, dh3...)

	if recipientOPK != nil {
		dh4, err := dh(recipientOPK.Private[:], ephemeralPublic)  // OPK_B × EK_A
		if err != nil { return nil, err }
		input = append(input, dh4...)
	}

	return kdf(input, "InstagramClone_X3DH_v1"), nil
}

func dh(private, public []byte) ([]byte, error) {
	if len(public) != 32 {
		return nil, errors.New("x3dh: public key must be 32 bytes")
	}
	return curve25519.X25519(private, public)
}

func kdf(input []byte, info string) []byte {
	salt := make([]byte, 32)
	h := hkdf.New(sha256.New, input, salt, []byte(info))
	out := make([]byte, 32)
	_, _ = h.Read(out)
	return out
}

func EncodeKey(key []byte) string {
	return base64.StdEncoding.EncodeToString(key)
}

func DecodeKey(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
