package main

import (
	"database/sql"
	"gator/internal/config"
	"gator/internal/database"
	"log"
	"os"

	_ "github.com/lib/pq"
)

type state struct {
	cfg *config.Config
	db  *database.Queries
}

func main() {
	c, err := config.Read()
	if err != nil {
		log.Fatalf("error reading config: %v", err)
	}

	db, err := sql.Open("postgres", c.Db_url)
	if err != nil {
		log.Fatalf("error connecting to db: %v", err)
	}

	programState := &state{cfg: &c, db: database.New(db)}
	commands := commands{registeredCommands: make(map[string]func(*state, command) error)}
	commands.register("login", handlerLogin)
	commands.register("register", handlerRegister)
	commands.register("reset", handlerReset)
	commands.register("users", handlerUsers)
	commands.register("agg", handlerAgg)
	commands.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	commands.register("feeds", handlerFeed)
	commands.register("follow", middlewareLoggedIn(handlerFollow))
	commands.register("following", middlewareLoggedIn(handlerFollowing))
	commands.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	commands.register("browse", middlewareLoggedIn(handlerBrowse))

	args := os.Args[1:]
	if len(args) < 1 {
		log.Fatalf("no command provided")
	}

	cmd := command{name: args[0], args: args[1:]}
	err = commands.run(programState, cmd)
	if err != nil {
		log.Fatalf("error running command: %v", err)
	}
}
