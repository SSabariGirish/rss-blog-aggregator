

# Gator – RSS Feed Aggregator

A command-line RSS feed aggregator written in **Go**, backed by **PostgreSQL**. Gator allows users to register, manage RSS feeds, follow feeds, continuously scrape new posts, and browse collected content directly from the terminal.

## Features

- **User management:** Register users, log in, and manage user records.
- **Feed management:** Add and list RSS feeds.
- **Feed following:** Follow and unfollow feeds, with each user's subscriptions tracked independently.
- **Automated aggregation:** Continuously fetch RSS feeds at configurable intervals.
- **Post storage:** Save feed entries to PostgreSQL while avoiding duplicate posts.
- **Personalised browsing:** Browse recent posts from feeds followed by the logged-in user.

## Tech Stack

- **Language:** Go
- **Database:** PostgreSQL
- **Database queries:** sqlc
- **Migrations:** Goose
- **PostgreSQL driver:** lib/pq
- **RSS parsing:** Go's `encoding/xml` package

## Prerequisites

Make sure you have the following installed:

- Go
- PostgreSQL
- [Goose](https://github.com/pressly/goose)
- [sqlc](https://sqlc.dev/)

## Installation

Clone the repository:

```bash
git clone https://github.com/SSabariGirish/blog-aggregator.git
cd gator
```

Install dependencies:

```bash
go mod download
```

## Configuration

Gator requires a PostgreSQL connection string.

Create a `.gatorconfig.json` file in your home directory:

```json
{
  "db_url": "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable",
  "current_user_name": ""
}
```

Update the connection string to match your PostgreSQL credentials and database configuration.

Create the database:

```sql
CREATE DATABASE gator;
```

Run database migrations from the schema directory:

```bash
cd sql/schema
goose postgres "postgres://postgres:postgres@localhost:5432/gator" up
```

Return to the project root:

```bash
cd ../..
```

## Usage

Run Gator using:

```bash
go run . <command> [arguments]
```

### User Commands

| Command | Description |
|---|---|
| `register <username>` | Register a new user |
| `login <username>` | Log in as an existing user |
| `users` | List all registered users |
| `reset` | Delete all users and associated data |

### Feed Commands

| Command | Description |
|---|---|
| `addfeed <name> <url>` | Add a feed and automatically follow it |
| `feeds` | List all registered feeds |
| `follow <url>` | Follow an existing feed |
| `unfollow <url>` | Unfollow a feed |
| `following` | List feeds followed by the current user |

### Aggregation Commands

| Command | Description |
|---|