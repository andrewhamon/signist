package models

import (
	"encoding/base64"
	"github.com/andrewhamon/signist/Godeps/_workspace/src/github.com/martini-contrib/binding"
	"github.com/andrewhamon/signist/Godeps/_workspace/src/golang.org/x/crypto/ssh"
	"net/http"
	"time"
)

type Signature struct {
	ID        *int       `json:"id,omitempty"`
	MessageID *int       `json:"message_id,omitempty" db:"message_id" binding:"required"`
	Message   *Message   `json:"-"`
	Format    *string    `json:"format" binding:"required"`
	Blob      *string    `json:"blob" binding:"required"`
	RawBlob   []byte     `json:"-"`
	Key       *PublicKey `json:"key"`
	CreatedAt *time.Time `json:"created_at,omitempty" db:"created_at"`
}

func (sig *Signature) ValidateBlob() *binding.Error {
	rawBlob, err := base64.StdEncoding.DecodeString(*sig.Blob)
	if err != nil {
		return &binding.Error{
			FieldNames:     []string{"signature.blob"},
			Classification: "InvalidInputError",
			Message:        "Blob could not be decoded as base64",
		}
	} else {
		sig.RawBlob = rawBlob
		return nil
	}
}

func (sig *Signature) ValidateSignature() *binding.Error {
	ghKeys := sig.Message.GithubKeys

	matchFound := false
	for _, k := range ghKeys {
		err := k.Verify(sig.Message.RawBlob, &ssh.Signature{Format: *sig.Format, Blob: sig.RawBlob})
		if err == nil {
			sig.Key = &PublicKey{k}
			matchFound = true
		}
	}

	if !matchFound {
		return &binding.Error{
			FieldNames:     []string{"key", "blob", "message.blob"},
			Classification: "SignatureInvalidError",
			Message:        "Key could not verify signature against message.blob",
		}
	} else {
		return nil
	}
}

func (sig *Signature) Validate(errors binding.Errors, req *http.Request) binding.Errors {
	var err *binding.Error

	if err = sig.ValidateBlob(); err != nil {
		return append(errors, *err)
	}

	if err = sig.ValidateSignature(); err != nil {
		return append(errors, *err)
	}

	return errors
}
