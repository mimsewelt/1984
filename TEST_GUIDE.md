# Instagram Clone — Test-Guide

## Voraussetzungen
- Go 1.23+
- `make` (optional, aber empfohlen)

Kein Docker nötig für Unit-Tests — alle Tests laufen ohne Datenbank.

---

## Tests ausführen

### Alle Tests auf einmal
```bash
make test-all
```

### Einzelne Services
```bash
make test-auth       # Auth Service (Register, Login, Refresh, bcrypt)
make test-gateway    # JWT Middleware (valid, expired, alg:none attack)
make test-messaging  # Signal Protocol X3DH Crypto
```

### Mit Race-Detector (empfohlen vor jedem Commit)
```bash
make test-auth       # -race ist bereits aktiviert
```

### Coverage-Report
```bash
make test-coverage
# Öffnet services/auth/coverage.html im Browser
```

### Ohne Make
```bash
cd services/auth     && go test ./... -v -race
cd services/gateway  && go test ./... -v -race
cd services/messaging && go test ./... -v -race
```

---

## Was wird getestet?

### Auth Service (`services/auth`)
| Test | Was geprüft wird |
|---|---|
| `TestRegister_Success` | User wird erstellt, Tokens werden zurückgegeben |
| `TestRegister_DuplicateEmail` | `ErrUserExists` bei doppelter Email |
| `TestLogin_WrongPassword` | `ErrInvalidCredentials` — kein Hinweis ob Email existiert |
| `TestLoginHandler_ErrorMessageIsGeneric` | Schutz vor User-Enumeration: gleiche Fehlermeldung bei falschem Passwort und unbekannter Email |
| `TestRefresh_RotatesToken` | Neues Token-Paar nach Refresh |
| `TestRefresh_OldTokenRejectedAfterRotation` | Replay-Angriff schlägt fehl |
| `TestPasswordHash_NotStoredPlaintext` | Passwort wird als bcrypt-Hash gespeichert |
| `TestUserID_IsUUID` | User-IDs sind valide UUIDs |

### Gateway Middleware (`services/gateway`)
| Test | Was geprüft wird |
|---|---|
| `TestAuthenticate_ValidToken_Passes` | Valides JWT kommt durch |
| `TestAuthenticate_InjectsUserIDIntoContext` | `user_id` landet im Context |
| `TestAuthenticate_ExpiredToken_Returns401` | Abgelaufene Tokens werden abgelehnt |
| `TestAuthenticate_WrongSecret_Returns401` | Falsches Signing-Secret → 401 |
| `TestAuthenticate_AlgNone_Rejected` | **Sicherheits-Test:** `alg:none`-Angriff wird blockiert |

### Signal Protocol Crypto (`services/messaging`)
| Test | Was geprüft wird |
|---|---|
| `TestX3DH_SenderRecipientAgreeOnSameSecret` | Sender und Empfänger berechnen identisches Shared Secret |
| `TestX3DH_DifferentSessions_DifferentSecrets` | Jede Session hat ein einzigartiges Secret |
| `TestX3DH_TamperedSPKSignature_Rejected` | Manipulierte Keys werden erkannt |
| `TestX3DH_WithoutOPK_StillAgreesOnSecret` | Funktioniert auch wenn alle One-Time PreKeys aufgebraucht sind |
| `TestSignedPreKey_TamperedSPK_FailsVerification` | Manipulierter SPK schlägt Signaturprüfung fehl |

---

## Wichtige Sicherheitsprinzipien im Code

**Kein Timing-Angriff bei Login:** bcrypt-Vergleich läuft immer durch, auch bei unbekannter Email.

**User Enumeration verhindert:** "invalid email or password" — nie "user not found".

**Refresh Token Rotation:** Jedes Token kann nur einmal verwendet werden. Replay = 401.

**`alg:none`-Angriff blockiert:** Middleware prüft explizit auf HMAC-Signaturmethode.

**Passwörter nie im Klartext:** bcrypt mit Cost 12 — ca. 300ms pro Hash (bewusst langsam).