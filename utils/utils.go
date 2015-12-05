package utils

import (
	"bytes"
	"encoding/base64"
	"github.com/andrewhamon/signist/github"
	"github.com/andrewhamon/signist/models"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"log"
	"net"
	"os"
)

// Convert agent.Key to ssh.PublicKey
func keyToPubKey(in *agent.Key) (out ssh.PublicKey, err error) {
	out, _, _, _, err = ssh.ParseAuthorizedKey([]byte(in.String()))
	return out, err
}

// Return a slice containing all the keys the agent knows about
func agentKeys() (pubKeys []ssh.PublicKey) {
	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		log.Printf("Error connecting to SSH agent: %s\n", err.Error())
		return []ssh.PublicKey{}
	}

	defer conn.Close()

	ag := agent.NewClient(conn)

	keys, err := ag.List()
	if err != nil {
		log.Printf("Error listing known identities in SSH agent: %s\n", err.Error())
		return []ssh.PublicKey{}
	}

	pubKeys = make([]ssh.PublicKey, 0, len(keys))

	for _, key := range keys {
		pubKey, err := keyToPubKey(key)
		if err == nil {
			pubKeys = append(pubKeys, pubKey)
		}
	}

	return pubKeys
}

// Check if two keys are equal
func KeysEq(first, second ssh.PublicKey) bool {
	if (first == nil) || (second == nil) {
		return false
	}
	return (first.Type() == second.Type()) && bytes.Equal(first.Marshal(), second.Marshal())
}

// Check if a slice of keys contains the given key
func KeyInSlice(key ssh.PublicKey, keys []ssh.PublicKey) bool {
	for _, k := range keys {
		if KeysEq(k, key) {
			return true
		}
	}
	return false
}

// Finds the keys common to the user/org and the local SSH agent
func commonKeys(login string) (keys []ssh.PublicKey) {
	user, err := github.UserFor(login)
	if err != nil {
		return []ssh.PublicKey{}
	}

	aks := agentKeys()
	gks := github.GithubKeysFor(user)

	if len(aks) > len(gks) {
		keys = make([]ssh.PublicKey, 0, len(aks))
	} else {
		keys = make([]ssh.PublicKey, 0, len(gks))
	}

	for _, k := range aks {
		if KeyInSlice(k, gks) {
			keys = append(keys, k)
		}
	}

	return keys
}

func PubKeyToString(key ssh.PublicKey) string {
	return key.Type() + " " + base64.StdEncoding.EncodeToString(key.Marshal())
}

// Sign data using any keys that can be found localy and remotely for the given user or org
func Sign(name string, data []byte) (sigs []*models.Signature) {
	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		log.Printf("Error connecting to SSH agent: %s\n", err.Error())
		return []*models.Signature{}
	}

	defer conn.Close()

	ag := agent.NewClient(conn)
	keys := commonKeys(name)
	sigs = make([]*models.Signature, 0, len(keys))

	for _, key := range keys {
		sig, err := ag.Sign(key, data)
		if err != nil {
			log.Printf("Error signing data with key %q: %s\n", key.Marshal(), err.Error())
		} else {
			blob := base64.StdEncoding.EncodeToString(sig.Blob)
			sigs = append(sigs, &models.Signature{Blob: &blob, Format: &sig.Format, Key: &models.PublicKey{key}})
		}
	}

	return sigs
}
