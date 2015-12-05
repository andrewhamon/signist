package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/andrewhamon/signist/Godeps/_workspace/src/gopkg.in/alecthomas/kingpin.v2"
	"github.com/andrewhamon/signist/models"
	"github.com/andrewhamon/signist/utils"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

func main() {
	name := kingpin.Arg("name", "Name of a github user or organization to sign as.").Required().String()
	title := kingpin.Arg("title", "Title for this signed message").Default(time.Now().Format("Mon-Jan-2-150405-MST")).String()
	kingpin.Parse()

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Error reading from standard input: %s\n", err.Error())
	}

	sigs := utils.Sign(*name, data)

	b64Data := base64.StdEncoding.EncodeToString(data)

	message := models.Message{GithubLogin: name, Blob: &b64Data, Title: title}
	message.Signatures = sigs
	payload, err := json.Marshal(message)
	if err != nil {
		log.Fatalln(err)
	}

	res, err := postToApi(*name, *title, payload)

	if err != nil {
		log.Fatalf("Error sending data to server: %s\n", err.Error())
	}

	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		body, _ := ioutil.ReadAll(res.Body)
		log.Fatalf("%s %s\n", res.StatusCode, body)
	}
}

func apiUrl() *url.URL {
	rawurl := os.Getenv("SIGNIST_API_URL")
	if len(rawurl) == 0 {
		rawurl = "https://api.signist.org"
	}

	parsedurl, err := url.Parse(rawurl)

	if err != nil {
		panic("Could not parse URL " + rawurl + "\n" + err.Error())
	}

	return parsedurl
}

func postToApi(login string, title string, payload []byte) (*http.Response, error) {
	destUrl := apiUrl()
	destUrl.Path = path.Join(login)
	return http.Post(destUrl.String(), "application/json", bytes.NewReader(payload))
}
