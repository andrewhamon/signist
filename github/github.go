package github

import (
	"github.com/andrewhamon/signist/Godeps/_workspace/src/github.com/google/go-github/github"
	"github.com/andrewhamon/signist/Godeps/_workspace/src/golang.org/x/crypto/ssh"
	"log"
)

// Global github client
var client = github.NewClient(nil)

// Return a github user for a particular login
// or exit on failure
func UserFor(login string) (*github.User, error) {
	user, _, err := client.Users.Get(login)
	if err != nil {
		log.Printf("Couldn't find a Github user or organization for %q: %s", login, err.Error())

	}
	return user, err
}

func isUser(user *github.User) bool {
	return *(user.Type) == "User"
}

func isOrg(user *github.User) bool {
	return *(user.Type) == "Organization"
}

// Return a slice containing all the publicly accessible organization admins
func orgAdmins(org string) (admins []github.User) {
	admins, _, err := client.Organizations.ListMembers(org, &github.ListMembersOptions{Role: "admin"})
	if err != nil {
		log.Printf("Error getting administrators for %q: %s\n", org, err.Error())
		return []github.User{}
	}
	return admins
}

func GithubKeysForUser(user string) (pubKeys []ssh.PublicKey) {
	keys, _, err := client.Users.ListKeys(user, nil)
	if err != nil {
		log.Printf("Error getting public keys for github user %q: %s\n", user, err.Error())
		return []ssh.PublicKey{}
	}

	pubKeys = make([]ssh.PublicKey, 0, len(keys))

	for _, key := range keys {
		pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(*key.Key))
		if err == nil {
			pubKeys = append(pubKeys, pubKey)
		}
	}
	return pubKeys
}

// Return all the public keys for all the admins of an org
func GithubKeysForOrg(org string) (pubKeys []ssh.PublicKey) {
	admins := orgAdmins(org)
	results := make(chan []ssh.PublicKey)
	pubKeys = []ssh.PublicKey{}

	for _, admin := range admins {
		admin := admin
		go func() {
			results <- GithubKeysForUser(*admin.Login)
		}()
	}

	for i := 0; i < len(admins); i++ {
		keys := <-results
		pubKeys = append(pubKeys, keys...)
	}

	return pubKeys
}

// Determine of a user is a user or organization and return the users keys
// or the keys for all admins of the org
func GithubKeysFor(user *github.User) (pubKeys []ssh.PublicKey) {
	if isUser(user) {
		return GithubKeysForUser(*user.Login)
	}

	if isOrg(user) {
		return GithubKeysForOrg(*user.Login)
	}

	log.Printf("Github identity %q is not a user or organization.\n", *user.Login)
	return []ssh.PublicKey{}
}
