
package main

import (
	"context"
    "encoding/xml"
    "fmt"
    "html"
    "io"
    "net/http"
	"database/sql"
	"log"
	"os"

	"github.com/SSabariGirish/gator/internal/config"
	"github.com/SSabariGirish/gator/internal/database"
	_ "github.com/lib/pq"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type RSSFeed struct {
    Channel RSSChannel `xml:"channel"`
}

type RSSChannel struct {
    Title       string    `xml:"title"`
    Link        string    `xml:"link"`
    Description string    `xml:"description"`
    Items       []RSSItem `xml:"item"`
}

type RSSItem struct {
    Title       string `xml:"title"`
    Link        string `xml:"link"`
    Description string `xml:"description"`
    PubDate     string `xml:"pubDate"`
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
    req, err := http.NewRequestWithContext(
        ctx,
        http.MethodGet,
        feedURL,
        nil,
    )
    if err != nil {
        return nil, fmt.Errorf("couldn't create request: %w", err)
    }

    req.Header.Set("User-Agent", "gator")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("couldn't fetch feed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return nil, fmt.Errorf("unexpected HTTP status: %s", resp.Status)
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("couldn't read response body: %w", err)
    }

    feed := &RSSFeed{}
    err = xml.Unmarshal(body, feed)
    if err != nil {
        return nil, fmt.Errorf("couldn't parse RSS feed: %w", err)
    }

    feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
    feed.Channel.Description = html.UnescapeString(feed.Channel.Description)

    for i := range feed.Channel.Items {
        feed.Channel.Items[i].Title = html.UnescapeString(feed.Channel.Items[i].Title)
        feed.Channel.Items[i].Description = html.UnescapeString(feed.Channel.Items[i].Description)
    }

    return feed, nil
}

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("error reading config: %v", err)
	}

	dbURL := cfg.DbURL

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("error connecting to db: %v", err)
	}
	defer db.Close()

	dbQueries := database.New(db)

	programState := &state{
		db:  dbQueries,
		cfg: &cfg,
	}

	cmds := commands{
		handlers: make(map[string]func(*state, command) error),
	}

	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", handlerAgg)
	cmds.register("feeds", handlerFeeds)
	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("following", middlewareLoggedIn(handlerFollowing))
	cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	cmds.register("browse", middlewareLoggedIn(handlerBrowse))

	if len(os.Args) < 2 {
		log.Fatal("Usage: cli <command> [args...]")
	}

	cmdName := os.Args[1]
	cmdArgs := os.Args[2:]

	err = cmds.run(programState, command{
		name: cmdName,
		args: cmdArgs,
	})
	if err != nil {
		log.Fatal(err)
	}
}
