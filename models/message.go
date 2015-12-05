package models

import (
	"encoding/base64"
	"github.com/andrewhamon/signist/github"
	"github.com/martini-contrib/binding"
	"golang.org/x/crypto/ssh"
	"net/http"
	"regexp"
	"time"
)

var slugRegex = regexp.MustCompile(`\A[a-zA-Z\d\-\_]+\z`)

type Message struct {
	ID          *int            `json:"id,omitempty"`
	GithubLogin *string         `json:"github_login" binding:"required"`
	GithubID    *int            `json:"-" db:"github_id"`
	GithubKeys  []ssh.PublicKey `json:"-"`
	Title       *string         `json:"title" binding:"required"`
	Blob        *string         `json:"blob" binding:"required"`
	RawBlob     []byte          `json:"-"`
	Signatures  []*Signature    `json:"signatures" binding:"required"`
	CreatedAt   *time.Time      `json:"created_at,omitempty" db:"created_at"`
}

func (message *Message) ValidateGithubLogin() *binding.Error {
	ghUser, err := github.UserFor(*message.GithubLogin)
	message.GithubID = ghUser.ID
	if err != nil {
		return &binding.Error{
			FieldNames:     []string{"github_login"},
			Classification: "DoesNotExistError",
			Message:        "The specified github user could not be found",
		}
	}
	message.GithubKeys = github.GithubKeysFor(ghUser)
	return nil
}

func (message *Message) ValidateTitle() *binding.Error {
	if !slugRegex.MatchString(*message.Title) {
		return &binding.Error{
			FieldNames:     []string{"title"},
			Classification: "InvalidInputError",
			Message:        "Title may only conaint letters, digits, hyphens, and underscores",
		}
	} else {
		return nil
	}
}

func (message *Message) ValidateBlob() *binding.Error {
	blob, err := base64.StdEncoding.DecodeString(*message.Blob)
	if err != nil {
		return &binding.Error{
			FieldNames:     []string{"message.blob"},
			Classification: "InvalidInputError",
			Message:        "Blob could not be decoded as base64",
		}
	} else {
		message.RawBlob = blob
		return nil
	}
}

func (message *Message) ValidateSignaturesLength() *binding.Error {
	if len(message.Signatures) == 0 {
		return &binding.Error{
			FieldNames:     []string{"signatures"},
			Classification: "",
			Message:        "There must be at least one signature for a message",
		}
	} else {
		return nil
	}
}

func (message *Message) ValidateSignatures() binding.Errors {
	results := make(chan binding.Errors, len(message.Signatures))

	for _, sig := range message.Signatures {
		sig := sig
		sig.Message = message
		go func() {
			results <- sig.Validate(binding.Errors{}, nil)
		}()
	}

	errors := binding.Errors{}
	for i := 0; i < len(message.Signatures); i++ {
		errors = append(errors, <-results...)
	}
	return errors
}

func (message *Message) Validate(errors binding.Errors, req *http.Request) binding.Errors {
	var err *binding.Error

	if err = message.ValidateGithubLogin(); err != nil {
		return append(errors, *err)
	}

	if err = message.ValidateTitle(); err != nil {
		return append(errors, *err)
	}

	if err = message.ValidateBlob(); err != nil {
		return append(errors, *err)
	}

	if err = message.ValidateSignaturesLength(); err != nil {
		return append(errors, *err)
	}

	errors = append(errors, message.ValidateSignatures()...)
	return errors
}
