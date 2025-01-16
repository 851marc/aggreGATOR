package main

import (
	"context"
	"database/sql"
	"fmt"

	"strconv"
	"strings"
	"time"

	"github.com/851marc/aggreGATOR/internal/database"
	"github.com/google/uuid"
)

type command struct {
	name string
	args []string
}
type commands struct {
	registeredCommands map[string]func(*state, command) error
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.registeredCommands[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	f, ok := c.registeredCommands[cmd.name]
	if !ok {
		return fmt.Errorf("command %v not found", cmd.name)
	}
	return f(s, cmd)
}

func handlerLogin(s *state, c command) error {
	if len(c.args) != 1 {
		return fmt.Errorf("login requires 1 argument")
	}

	u, err := s.db.GetUser(context.Background(), c.args[0])
	if err != nil || u == (database.User{}) {
		return fmt.Errorf("error getting user: %v", err)
	}

	err = s.cfg.SetUser(c.args[0])
	if err != nil {
		return fmt.Errorf("error setting user: %v", err)
	}
	fmt.Printf("User %v set \n", c.args[0])
	return nil
}

func handlerRegister(s *state, c command) error {
	if len(c.args) != 1 {
		return fmt.Errorf("register requires 1 argument")
	}

	u, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      c.args[0],
	})

	if err != nil {
		return fmt.Errorf("error creating user: %v", err)
	}

	s.cfg.SetUser(c.args[0])
	fmt.Printf("User %v registered \n", u)

	return nil
}

func handlerReset(s *state, c command) error {
	if len(c.args) != 0 {
		return fmt.Errorf("reset does not require arguments")
	}

	err := s.db.Reset(context.Background())

	if err != nil {
		return fmt.Errorf("error resetting users: %v", err)
	}
	fmt.Println("Users reset")
	return nil
}

func handlerUsers(s *state, c command) error {
	if len(c.args) != 0 {
		return fmt.Errorf("users does not require arguments")
	}

	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error getting users: %v", err)
	}

	for _, u := range users {
		if u.Name == s.cfg.Current_user_name {
			fmt.Printf("%v (current)\n", u.Name)
			continue
		}
		fmt.Printf("%v\n", u.Name)
	}

	return nil
}

func handlerAgg(s *state, c command) error {
	if len(c.args) != 1 {
		return fmt.Errorf("agg require 1 argument")
	}

	time_between_reqs, err := time.ParseDuration(c.args[0])
	if err != nil {
		return fmt.Errorf("error parsing duration: %v", err)
	}

	fmt.Printf("Collecting feeds every  %v\n", time_between_reqs)
	ticker := time.NewTicker(time_between_reqs)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
}

func handlerAddFeed(s *state, c command, u database.User) error {
	if len(c.args) != 2 {
		return fmt.Errorf("add-feed requires 2 arguments")
	}

	dbFeed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      c.args[0],
		Url:       c.args[1],
		UserID:    u.ID,
	})

	if err != nil {
		return fmt.Errorf("error creating feed: %v", err)
	}

	_, err = s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    u.ID,
		FeedID:    dbFeed.ID,
	})

	if err != nil {
		return fmt.Errorf("error following feed: %v", err)
	}

	fmt.Printf("Feed %v added \n", dbFeed)
	return nil
}

func handlerFeed(s *state, c command) error {
	if len(c.args) != 0 {
		return fmt.Errorf("feed does not require arguments")
	}

	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("error getting feeds: %v", err)
	}

	for _, f := range feeds {
		u, err := s.db.GetUserById(context.Background(), f.UserID)
		if err != nil || u == (database.User{}) {
			return fmt.Errorf("error getting user: %v", err)
		}

		fmt.Printf("%v %v %v\n", f.Name, f.Url, u.Name)
	}

	return nil
}

func handlerFollow(s *state, c command, u database.User) error {
	if len(c.args) != 1 {
		return fmt.Errorf("follow requires 1 argument")
	}

	f, err := s.db.GetFeedByUrl(context.Background(), c.args[0])
	if err != nil || f == (database.Feed{}) {
		return fmt.Errorf("error getting feed: %v", err)
	}

	cf, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		UserID: u.ID,
		FeedID: f.ID,
	})

	if err != nil {
		return fmt.Errorf("error following feed: %v", err)
	}

	fmt.Printf("%v followed %v \n", cf.UserName, cf.FeedName)
	return nil
}

func handlerFollowing(s *state, c command, u database.User) error {
	if len(c.args) != 0 {
		return fmt.Errorf("following does not require arguments")
	}

	follows, err := s.db.GetFeedFollowsForUser(context.Background(), u.ID)
	if err != nil {
		return fmt.Errorf("error getting follows: %v", err)
	}

	fmt.Printf("%v is following:\n", u.Name)
	for _, f := range follows {
		fmt.Printf("  %v\n", f.FeedName)
	}

	return nil
}

func handlerUnfollow(s *state, c command, u database.User) error {
	if len(c.args) != 1 {
		return fmt.Errorf("unfollow requires 1 argument")
	}

	f, err := s.db.GetFeedByUrl(context.Background(), c.args[0])
	if err != nil || f == (database.Feed{}) {
		return fmt.Errorf("error getting feed: %v", err)
	}

	err = s.db.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{UserID: u.ID, FeedID: f.ID})
	if err != nil {
		return fmt.Errorf("error unfollowing feed: %v", err)
	}

	fmt.Printf("%v unfollowed %v \n", u.Name, f.Name)
	return nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		if s.cfg.Current_user_name == "" {
			return fmt.Errorf("not logged in")
		}

		u, err := s.db.GetUser(context.Background(), s.cfg.Current_user_name)
		if err != nil || u == (database.User{}) {
			return fmt.Errorf("error getting user: %v", err)
		}

		return handler(s, cmd, u)
	}
}

func scrapeFeeds(s *state) error {
	feed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return fmt.Errorf("error getting next feed to fetch: %v", err)
	}

	fmt.Println("Fetching feed ", feed)

	f, err := FetchFeed(context.Background(), feed.Url)
	if err != nil {
		return fmt.Errorf("error fetching feed %v: %v", feed, err)
	}

	fmt.Println("Fetched feed ", feed)

	err = s.db.MarkFeedFetched(context.Background(), feed.ID)
	if err != nil {
		return fmt.Errorf("error marking feed fetched: %v", err)
	}

	fmt.Println("Parsing feed items ", len(f.Channel.Item))

	for _, v := range f.Channel.Item {

		publishedAt, err := time.Parse(time.RFC1123Z, v.PubDate)

		fmt.Println("Creating post ", v.Title)

		_, err = s.db.CreatePost(context.Background(), database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Title:       v.Title,
			Url:         v.Link,
			Description: sql.NullString{String: v.Description, Valid: v.Description != ""},
			PublishedAt: sql.NullTime{Time: publishedAt, Valid: err != nil},
			FeedID:      feed.ID,
		})

		if err != nil {
			if strings.Contains(err.Error(), "duplicate key") {
				continue
			}
			return fmt.Errorf("error creating post: %v", err)
		}
	}

	return nil
}

func handlerBrowse(s *state, c command, u database.User) error {
	if len(c.args) > 1 {
		return fmt.Errorf("browse requires 0 or 1 arguments")
	}

	limit := int32(2)
	if len(c.args) == 1 {
		if t, err := strconv.Atoi(c.args[0]); err == nil {
			limit = int32(t)
		}

		posts, err := s.db.GetPostByUser(context.Background(), database.GetPostByUserParams{UserID: u.ID, Limit: limit})
		if err != nil {
			return fmt.Errorf("error getting posts: %v", err)
		}

		fmt.Printf("Posts for %v:\n", u.Name)
		for _, p := range posts {
			fmt.Printf("%v %v\n", p.Title, p.Description)
		}

		return nil
	}

	return nil
}
