// ------------------------
// stdlib_hash.go
// ------------------------

package main

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func InitHashFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "hash."

	// hash.MD5(s)
	Register(ns+"MD5", "hash", "s", "Erzeugt einen MD5-Hash.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("hash.MD5: Argument fehlt")
		}
		h := md5.Sum([]byte(ToString(args[0])))
		return StrVal(hex.EncodeToString(h[:]))
	})

	// hash.SHA1(s)
	Register(ns+"SHA1", "hash", "s", "Erzeugt einen SHA1-Hash (für Legacy-Kompatibilität).", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("hash.SHA1: Argument fehlt")
		}
		h := sha1.Sum([]byte(ToString(args[0])))
		return StrVal(hex.EncodeToString(h[:]))
	})

	// hash.SHA256(s)
	Register(ns+"SHA256", "hash", "s", "Erzeugt einen SHA256-Hash.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("hash.SHA256: Argument fehlt")
		}
		h := sha256.Sum256([]byte(ToString(args[0])))
		return StrVal(hex.EncodeToString(h[:]))
	})

	// hash.SHA512(s)
	Register(ns+"SHA512", "hash", "s", "Erzeugt einen SHA512-Hash.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("hash.SHA512: Argument fehlt")
		}
		h := sha512.Sum512([]byte(ToString(args[0])))
		return StrVal(hex.EncodeToString(h[:]))
	})

	// hash.HMAC(s, key, [algo])
	// algo: "sha256" (Standard), "sha512", "sha1", "md5"
	Register(ns+"HMAC", "hash", "s, key, [algo]", "Erzeugt einen HMAC. algo: sha256 (Standard), sha512, sha1, md5.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("hash.HMAC: s und key benötigt")
		}

		data := []byte(ToString(args[0]))
		key := []byte(ToString(args[1]))

		algo := "sha256"
		if len(args) >= 3 {
			algo = ToString(args[2])
		}

		var mac []byte
		switch algo {
		case "sha256":
			h := hmac.New(sha256.New, key)
			h.Write(data)
			mac = h.Sum(nil)
		case "sha512":
			h := hmac.New(sha512.New, key)
			h.Write(data)
			mac = h.Sum(nil)
		case "sha1":
			h := hmac.New(sha1.New, key)
			h.Write(data)
			mac = h.Sum(nil)
		case "md5":
			h := hmac.New(md5.New, key)
			h.Write(data)
			mac = h.Sum(nil)
		default:
			return ErrorVal("hash.HMAC: Unbekannter Algorithmus '" + algo + "'. Unterstützt: sha256, sha512, sha1, md5")
		}

		return StrVal(hex.EncodeToString(mac))
	})

	// hash.Bcrypt(pass, [cost])
	// cost: 10 (Standard), 4-31
	Register(ns+"Bcrypt", "hash", "pass, [cost]", "Erzeugt einen sicheren Bcrypt-Hash eines Passworts. cost: 4-31, Standard 10.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("hash.Bcrypt: Passwort fehlt")
		}

		cost := bcrypt.DefaultCost // 10
		if len(args) >= 2 {
			c := int(toNumVal(args[1]))
			if c >= bcrypt.MinCost && c <= bcrypt.MaxCost {
				cost = c
			}
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(ToString(args[0])), cost)
		if err != nil {
			return ErrorVal("hash.Bcrypt: " + err.Error())
		}

		return StrVal(string(hashed))
	})

	// hash.BcryptVerify(pass, hash)
	Register(ns+"BcryptVerify", "hash", "pass, hash", "Prüft ob ein Passwort zu einem Bcrypt-Hash passt.", func(args []Value) Value {
		if len(args) < 2 {
			return ErrorVal("hash.BcryptVerify: pass und hash benötigt")
		}

		err := bcrypt.CompareHashAndPassword([]byte(ToString(args[1])), []byte(ToString(args[0])))
		return BoolVal(err == nil)
	})

	// hash.File(path, [algo])
	// algo: "sha256" (Standard), "sha512", "sha1", "md5"
	Register(ns+"File", "hash", "path, [algo]", "Berechnet den Hash einer Datei. algo: sha256 (Standard), sha512, sha1, md5.", func(args []Value) Value {
		if len(args) < 1 {
			return ErrorVal("hash.File: Pfad fehlt")
		}

		path, errVal := absPathVal(args[0].Str)
		if errVal != nil {
			return *errVal
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return ErrorVal("hash.File: " + err.Error())
		}

		algo := "sha256"
		if len(args) >= 2 {
			algo = ToString(args[1])
		}

		switch algo {
		case "sha256":
			h := sha256.Sum256(data)
			return StrVal(hex.EncodeToString(h[:]))
		case "sha512":
			h := sha512.Sum512(data)
			return StrVal(hex.EncodeToString(h[:]))
		case "sha1":
			h := sha1.Sum(data)
			return StrVal(hex.EncodeToString(h[:]))
		case "md5":
			h := md5.Sum(data)
			return StrVal(hex.EncodeToString(h[:]))
		default:
			return ErrorVal("hash.File: Unbekannter Algorithmus '" + algo + "'")
		}
	})
}
