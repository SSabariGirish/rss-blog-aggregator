
package main

import (
	"context"
	"fmt"
	"time"
	"database/sql"
    "errors"
    "strconv"

	"github.com/lib/pq"
	"github.com/SSabariGirish/gator/internal/database"
	"github.com/google/uuid"
)

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("usage: %s <name>", cmd.name)
	}

	name := cmd.args[0]
	now := time.Now().UTC()

	user, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Name:      name,
	})
	if err != nil {
		return fmt.Errorf("couldn't create user: %w", err)
	}

	if err := s.cfg.SetUser(user.Name); err != nil {
		return fmt.Errorf("couldn't set current user: %w", err)
	}

	fmt.Println("User created successfully:")
	printUser(user)

	return nil
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("usage: %s <name>", cmd.name)
	}

	name := cmd.args[0]

	_, err := s.db.GetUser(context.Background(), name)
	if err != nil {
		return fmt.Errorf("couldn't find user: %w", err)
	}

	if err := s.cfg.SetUser(name); err != nil {
		return fmt.Errorf("couldn't set current user: %w", err)
	}

	fmt.Println("User switched successfully!")
	return nil
}

func printUser(user database.User) {
	fmt.Printf(" * ID:      %v\n", user.ID)
	fmt.Printf(" * Name:    %v\n", user.Name)
}

func handlerReset(s *state, cmd command) error {
    err := s.db.DeleteAllUsers(context.Background())
    if err != nil {
        return fmt.Errorf("couldn't reset database: %w", err)
    }

    fmt.Println("Database reset successfully!")
    return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't gets users: %w", err)
	}

	for _, user := range users {
		if user.Name == s.cfg.CurrentUserName {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}

	return nil
}

func handlerAgg(s *state, cmd command) error {
    if len(cmd.args) != 1 {
        return fmt.Errorf("usage: %s <time_between_reqs>", cmd.name)
    }

    timeBetweenRequests, err := time.ParseDuration(cmd.args[0])
    if err != nil {
        return fmt.Errorf("invalid duration: %w", err)
    }

    fmt.Printf("Collecting feeds every %v\n", timeBetweenRequests)

    ticker := time.NewTicker(timeBetweenRequests)
    defer ticker.Stop()

    for ; ; <-ticker.C {
        err := scrapeFeeds(s)
        if err != nil {
            fmt.Printf("Error scraping feeds: %v\n", err)
        }
    }
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
    if len(cmd.args) != 2 {
        return fmt.Errorf("usage: %s <name> <url>", cmd.name)
    }

    ctx := context.Background()
    name := cmd.args[0]
    url := cmd.args[1]
    now := time.Now().UTC()

    feed, err := s.db.CreateFeed(ctx, database.CreateFeedParams{
        ID:        uuid.New(),
        CreatedAt: now,
        UpdatedAt: now,
        Name:      name,
        Url:       url,
        UserID:    user.ID,
    })
    if err != nil {
        return fmt.Errorf("couldn't create feed: %w", err)
    }

	_, err = s.db.CreateFeedFollow(ctx, database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("couldn't follow feed: %w", err)
	}

    fmt.Println("Feed created successfully:")
    fmt.Printf(" * ID:         %v\n", feed.ID)
    fmt.Printf(" * Name:       %v\n", feed.Name)
    fmt.Printf(" * URL:        %v\n", feed.Url)
    fmt.Printf(" * User ID:    %v\n", feed.UserID)

    return nil
}

func handlerFeeds(s *state, cmd command) error {
    if len(cmd.args) != 0 {
        return fmt.Errorf("usage: %s", cmd.name)
    }

    feeds, err := s.db.GetFeeds(context.Background())
    if err != nil {
        return fmt.Errorf("couldn't get feeds: %w", err)
    }

    for _, feed := range feeds {
        fmt.Printf("Feed: %s\n", feed.FeedName)
        fmt.Printf("URL: %s\n", feed.Url)
        fmt.Printf("Created by: %s\n", feed.UserName)
        fmt.Println()
    }

    return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
    if len(cmd.args) != 1 {
        return fmt.Errorf("usage: %s <url>", cmd.name)
    }

    ctx := context.Background()

    feed, err := s.db.GetFeedByURL(ctx, cmd.args[0])
    if err != nil {
        return fmt.Errorf("couldn't get feed: %w", err)
    }

    now := time.Now().UTC()

    follow, err := s.db.CreateFeedFollow(ctx, database.CreateFeedFollowParams{
        ID:        uuid.New(),
        CreatedAt: now,
        UpdatedAt: now,
        UserID:    user.ID,
        FeedID:    feed.ID,
    })
    if err != nil {
        return fmt.Errorf("couldn't create feed follow: %w", err)
    }

    fmt.Printf("%s is now following %s\n", follow.UserName, follow.FeedName)

    return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
    if len(cmd.args) != 0 {
        return fmt.Errorf("usage: %s", cmd.name)
    }

    ctx := context.Background()

    follows, err := s.db.GetFeedFollowsForUser(ctx, user.ID)
    if err != nil {
        return fmt.Errorf("couldn't get feed follows: %w", err)
    }

    fmt.Printf("%s is following:\n", user.Name)

    for _, follow := range follows {
        fmt.Printf(" * %s\n", follow.FeedName)
    }

    return nil
}

func middlewareLoggedIn(
    handler func(s *state, cmd command, user database.User) error,
) func(*state, command) error {
    return func(s *state, cmd command) error {
        user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
        if err != nil {
            return fmt.Errorf("couldn't get current user: %w", err)
        }

        return handler(s, cmd, user)
    }
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
    if len(cmd.args) != 1 {
        return fmt.Errorf("usage: %s <url>", cmd.name)
    }

    ctx := context.Background()

    feed, err := s.db.GetFeedByURL(ctx, cmd.args[0])
    if err != nil {
        return fmt.Errorf("couldn't get feed: %w", err)
    }

    err = s.db.DeleteFeedFollow(ctx, database.DeleteFeedFollowParams{
        UserID: user.ID,
        FeedID: feed.ID,
    })
    if err != nil {
        return fmt.Errorf("couldn't unfollow feed: %w", err)
    }

    fmt.Printf("%s unfollowed %s\n", user.Name, feed.Name)

    return nil
}

func scrapeFeeds(s *state) error {
    ctx := context.Background()

    feed, err := s.db.GetNextFeedToFetch(ctx)
    if err != nil {
        return fmt.Errorf("couldn't get next feed: %w", err)
    }

    err = s.db.MarkFeedFetched(ctx, feed.ID)
    if err != nil {
        return fmt.Errorf("couldn't mark feed as fetched: %w", err)
    }

    fmt.Printf("Fetching feed: %s\n", feed.Name)

    rssFeed, err := fetchFeed(ctx, feed.Url)
    if err != nil {
        return fmt.Errorf("couldn't fetch feed %s: %w", feed.Name, err)
    }

    for _, item := range rssFeed.Channel.Items {
		publishedAt, err := parsePublishedAt(item.PubDate)
		if err != nil {
			fmt.Printf("Couldn't parse publication date for %q: %v\n", item.Title, err)
			publishedAt = sql.NullTime{}
		}

		description := sql.NullString{
			String: item.Description,
			Valid:  item.Description != "",
		}

		_, err = s.db.CreatePost(ctx, database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
			Title:       item.Title,
			Url:         item.Link,
			Description: description,
			PublishedAt: publishedAt,
			FeedID:      feed.ID,
		})

		if err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23505" {
				// This post already exists. That's expected; skip it.
				continue
			}

			fmt.Printf("Couldn't save post %q: %v\n", item.Title, err)
			continue
		}

		fmt.Printf("Saved post: %s\n", item.Title)
	}

    return nil
}

func handlerBrowse(s *state, cmd command, user database.User) error {
    if len(cmd.args) > 1 {
        return fmt.Errorf("usage: %s [limit]", cmd.name)
    }

    limit := 2

    if len(cmd.args) == 1 {
        parsedLimit, err := strconv.Atoi(cmd.args[0])
        if err != nil || parsedLimit < 1 {
            return fmt.Errorf("limit must be a positive number")
        }
        limit = parsedLimit
    }

    posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
        UserID: user.ID,
        Limit:  int32(limit),
    })
    if err != nil {
        return fmt.Errorf("couldn't get posts: %w", err)
    }

    for _, post := range posts {
        fmt.Printf("Title: %s\n", post.Title)
        fmt.Printf("URL: %s\n", post.Url)

        if post.Description.Valid {
            fmt.Printf("Description: %s\n", post.Description.String)
        }

        if post.PublishedAt.Valid {
            fmt.Printf("Published: %s\n", post.PublishedAt.Time.Format(time.RFC1123))
        }

        fmt.Println()
    }

    return nil
}

func parsePublishedAt(value string) (sql.NullTime, error) {
    if value == "" {
        return sql.NullTime{}, nil
    }

    layouts := []string{
        time.RFC3339,
        time.RFC3339Nano,
        time.RFC1123Z,
        time.RFC1123,
        time.RFC822Z,
        time.RFC822,
    }

    for _, layout := range layouts {
        parsed, err := time.Parse(layout, value)
        if err == nil {
            return sql.NullTime{
                Time:  parsed,
                Valid: true,
            }, nil
        }
    }

    return sql.NullTime{}, fmt.Errorf("unsupported publication date format: %q", value)
}