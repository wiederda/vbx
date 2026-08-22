#use crypt
' ============================
' Crypt Beispiele für VBX
' ============================

Print "=== Hashes ==="
Print "MD5:    " & MD5("Hello VBmini")
Print "SHA1:   " & SHA1("Hello VBmini")
Print "SHA256: " & SHA256("Hello VBmini")
Print "SHA512: " & SHA512("Hello VBmini")
Print ""

Print "=== Random / GUID ==="
Print "Random GUID:      " & crypt.GUID()
Print "RNGCryptoProvider: " & crypt.RNGCryptoProvider(16)
Print "Random String:    " & crypt.RandomString(12)
Print "Random Password: " & crypt.RandomPassword(12, True, True, True, True, True)
Print "Random Password: " & crypt.RandomPassword(12, True, True, False, True, True)
Print ""

Print "=== Base64 ==="
Dim b64
b64 = EncodeBase64("Hello VBmini")
Print "Encoded: " & b64
Print "Decoded: " & DecodeBase64(b64)
Print ""

Print "=== AES-GCM Encryption / Decryption ==="
Dim encrypted = crypt.AESEncrypt("Das ist ein geheimer Text!")
Print "Encrypted: " & encrypted
Dim decrypted = crypt.AESDecrypt(encrypted)
Print "Decrypted: " & decrypted
Print ""

Print "Encrypted (custom password):"
Dim encCustom = crypt.AESEncrypt("Das ist ein geheimer Text!", "AnderePasswort123!")
Print "Encrypted: " & encCustom
Dim decCustom = crypt.AESDecrypt(encCustom, "AnderePasswort123!")
Print "Decrypted: " & decCustom