package models

import (
	"encoding/base64"
	"errors"
	"golang.org/x/crypto/ssh"
)

type PublicKey struct {
	ssh.PublicKey
}

func (key PublicKey) MarshalJSON() ([]byte, error) {
	result := `"` + key.Type() + " " + base64.StdEncoding.EncodeToString(key.Marshal()) + `"`
	return []byte(result), nil
}

func (key *PublicKey) UnmarshalJSON(b []byte) error {
	if b[0] != byte('"') {
		return errors.New(`Expected to start with "`)
	}

	if b[len(b)-1] != byte('"') {
		return errors.New(`Expected to end with "`)
	}

	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(b[2 : len(b)-1])
	if err != nil {
		return err
	}

	*key = PublicKey{pubKey}
	return nil
}

func (key *PublicKey) Scan(src interface{}) error {
	if src == nil {
		return errors.New("PublicKey can not be nil")
	}

	var out ssh.PublicKey
	var err error

	switch src.(type) {
	case string:
		out, _, _, _, err = ssh.ParseAuthorizedKey([]byte(src.(string)))
	case []byte:
		out, _, _, _, err = ssh.ParseAuthorizedKey(src.([]byte))
	default:
		return errors.New("Can not convert that type to PublicKey")
	}

	*key = PublicKey{out}
	return err
}
