package main

import (
	"github.com/andrewhamon/signist/github"
	"github.com/andrewhamon/signist/models"
	"github.com/andrewhamon/signist/utils"
	"github.com/go-martini/martini"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	"log"
	"net/http"
	"os"
	"time"
)

type Message models.Message
type Signature models.Signature

type jsonError struct {
	Error string
}

func databaseString() string {
	dbstr := os.Getenv("DATABASE_URL")
	if len(dbstr) > 0 {
		return dbstr
	}

	return "user=signist dbname=signist sslmode=disable"
}

func main() {
	db, err := sqlx.Connect("postgres", databaseString())
	if err != nil {
		log.Fatalln(err)
	}

	m := martini.Classic()
	m.Use(render.Renderer())

	m.Get("/:login", func(params martini.Params, r render.Render) {
		user, err := github.UserFor(params["login"])
		if err != nil {
			r.JSON(http.StatusNotFound, jsonError{Error: "That github user could not be found"})
			return
		}

		messages := []*models.Message{}

		err = db.Select(&messages, "SELECT * FROM messages WHERE github_id = $1", user.ID)

		if err != nil {
			log.Println(err)
			r.JSON(http.StatusInternalServerError, err)
			return
		}

		for _, m := range messages {
			m := m

			m.Signatures = []*models.Signature{}
			err := db.Select(&m.Signatures, "SELECT * FROM signatures WHERE message_id = $1", m.ID)
			if err != nil {
				log.Println(err)
				r.JSON(http.StatusInternalServerError, err)
				return
			}
		}
		r.JSON(http.StatusOK, messages)
	})

	m.Post("/:login", binding.Bind(models.Message{}), func(message models.Message, params martini.Params, r render.Render) {
		user := params["login"]
		if user != *message.GithubLogin {
			r.JSON(http.StatusBadRequest, jsonError{Error: "Login in payload doesn't match login in URL"})
			return
		}

		tx := db.MustBegin()
		err := tx.QueryRowx(`INSERT INTO messages (github_id, title, blob, created_at) VALUES ($1, $2, $3, $4) RETURNING id, created_at`, message.GithubID, message.Title, message.Blob, time.Now()).StructScan(&message)

		if err != nil {
			tx.Rollback()
			log.Println(err)
			r.JSON(http.StatusBadRequest, err)
			return
		}

		for _, sig := range message.Signatures {
			err := tx.QueryRowx(`INSERT INTO signatures (message_id, format, blob, key, created_at) VALUES ($1, $2, $3, $4, $5) RETURNING id, message_id, created_at`, message.ID, sig.Format, sig.Blob, utils.PubKeyToString(*sig.Key), time.Now()).StructScan(sig)

			if err != nil {
				tx.Rollback()
				log.Println(err)
				r.JSON(http.StatusBadRequest, err)
				return
			}

		}

		if err := tx.Commit(); err != nil {
			tx.Rollback()
			r.JSON(http.StatusBadRequest, err)
			return
		}

		r.JSON(http.StatusOK, message)

	})

	m.Run()
}
