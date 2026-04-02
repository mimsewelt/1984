package crypto

// Signal Protocol – Extended Triple Diffie-Hellman (X3DH) key bundle.
//
// How it works:
//   1. Every user generates and publishes a key bundle to the server.
//   2. Sender fetches recipient's bundle, performs X3DH → shared secret.
//   3. Shared secret seeds a Double Ratchet (DR) session.
//   4. DR produces a new encryption key for every single message.
//   5. Server stores ONLY ciphertext — it cannot decrypt anything.
//
// Key types (all Curve25519 / Ed25519):
//   IK  — Identity Key   (long-term, changes only on account reset)
//   SPK — Signed PreKey  (rotated weekly, signed by IK for authenticity)
//   OPK — One-Time PreKey (consumed once per session, prevents replay)
//   EK  — Ephemeral Key  (generated fresh per X3DH exchange, by sender)

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
 OPKBatchSize    = 100  // pre-keys uploaded per batch
 SPKRotateDays   = 7    // signed pre-key rotation period
 OPKLowThreshold = 10   // server warns client to refill when below this
)

// KeyPair is a Curve25519 Diffie-Hellman key pair.
type KeyPair struct {
 Private [32]byte
 Public  [32]byte
}

// IdentityKeyPair holds both DH and signing keys for a user.
type IdentityKeyPair struct {
 DH      KeyPair          // used in X3DH
 Signing ed25519.PrivateKey // used to sign SPK
}

// PreKeyBundle is what the server publishes per user.
// The sender fetches this to initiate a session.
type PreKeyBundle struct {
 UserID         string    json:"user_id"
 IdentityKey    []byte    json:"ik"      // IK public (32 bytes)
 SignedPreKey   []byte    json:"spk"     // SPK public (32 bytes)
 SPKSignature   []byte    json:"spk_sig" // Ed25519 sig of SPK by IK
 SPKKeyID       uint32    json:"spk_id"
 OneTimePreKey  []byte    json:"opk"     // OPK public (32 bytes), may be nil
 OPKKeyID       uint32    json:"opk_id"
 PublishedAt    time.Time json:"published_at"
}

// GenerateDHKeyPair generates a fresh Curve25519 key pair.
func GenerateDHKeyPair() (KeyPair, error) {
 var priv [32]byte
 if _, err := rand.Read(priv[:]); err != nil {
  return KeyPair{}, err
 }
 // Clamp per RFC 7748.
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

// GenerateIdentityKeyPair creates the long-term identity key.
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

// GenerateSignedPreKey creates a SPK and signs its public bytes with the identity signing key.
func GenerateSignedPreKey(ik IdentityKeyPair, keyID uint32) (KeyPair, []byte, error) {
 spk, err := GenerateDHKeyPair()
 if err != nil {
  return KeyPair{}, nil, err
 }
 sig := ed25519.Sign(ik.Signing, spk.Public[:])
 return spk, sig, nil
}

// VerifySPKSignature checks the SPK was signed by the claimed identity key.
func VerifySPKSignature(ikPublicEd []byte, spkPublic, sig []byte) bool {
 return ed25519.Verify(ed25519.PublicKey(ikPublicEd), spkPublic, sig)
}

// GenerateOPKBatch generates a batch of one-time pre-keys.
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

// X3DHSender performs the X3DH key agreement as the initiating party.
// Returns the 32-byte shared master secret and the ephemeral public key to send.
//
// Calculation (Signal spec):
//   DH1 = DH(IK_A, SPK_B)
//   DH2 = DH(EK_A, IK_B)
//   DH3 = DH(EK_A, SPK_B)
//   DH4 = DH(EK_A, OPK_B)  // only if OPK present
//   SK  = KDF(DH1  DH2  DH3 [|| DH4])
func X3DHSender(
 senderIK IdentityKeyPair,