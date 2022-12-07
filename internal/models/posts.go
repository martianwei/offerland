package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Post struct {
	PostID    uuid.UUID           `json:"post_id"`
	Title     string              `json:"title"`
	AddResult bool                `json:"add_result"`
	Body      string              `json:"body"`
	CreatedAt time.Time           `json:"created_at"`
	UserID    uuid.UUID           `json:"user_id"`
	filter    map[string][]string `json:"-"`
}

type PostModel struct {
	DB *sql.DB
}

func (m PostModel) GetPostByID(postID uuid.UUID) (Post, error) {
	post := Post{}
	var query string
	cols := []string{
		"post_id",
		"title",
		"add_result",
		"body",
		"created_at",
		"user_id",
	}
	query = fmt.Sprintf(`
		SELECT %s
		FROM posts
		WHERE post_id = $1
	`, strings.Join(cols, ","))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, postID).Scan(
		&post.PostID,
		&post.Title,
		&post.AddResult,
		&post.Body,
		&post.CreatedAt,
		&post.UserID,
	)

	if err != nil {
		return post, err
	}

	return post, nil
}

func (m PostModel) CheckPostIsMine(post *Post) (bool, error) {
	var postOwner uuid.UUID

	var query = `
		SELECT (user_id)
		FROM posts 
		WHERE post_id = $1
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, post.PostID).Scan(&postOwner)

	if err != nil {
		return false, err
	}

	if postOwner != post.UserID {
		return false, nil
	} else {
		return true, nil
	}
}

func (m PostModel) Delete(post *Post) error {

	var query = `
		DELETE FROM posts 
		WHERE post_id = $1
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.QueryContext(ctx, query, post.PostID)

	if err != nil {
		return err
	}

	return nil
}

func (m PostModel) Upsert(post *Post) error {
	var query string
	cols := []string{
		"post_id",
		"title",
		"add_result",
		"body",
		"user_id",
	}
	exCols := []string{}
	for _, col := range cols {
		exCols = append(exCols, "EXCLUDED."+col)
	}
	jsonStr, err := json.Marshal(post)
	if err != nil {
		return err
	}
	setMap := map[string]any{}
	if err := json.Unmarshal(jsonStr, &setMap); err != nil {
		return err
	}
	index := 1
	var args []any
	var values []string
	for _, col := range cols {
		values = append(values, fmt.Sprintf("$%d", index))
		args = append(args, setMap[col])
		index++
	}
	query = fmt.Sprintf(`
		INSERT INTO posts (%s)
		VALUES (%s)
		ON CONFLICT (post_id)
		DO UPDATE SET (%s) = (%s);
		`, strings.Join(cols, ","),
		strings.Join(values, ","),
		strings.Join(cols, ","),
		strings.Join(exCols, ","),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err = m.DB.ExecContext(ctx, query, args...)

	return err
}

func (m PostModel) GetAllPosts(filter map[string][]string) ([]Post, error) {
	var query string
	cols := []string{
		"post_id",
		"title",
		"add_result",
		"body",
		"created_at",
		"user_id",
	}
	query = fmt.Sprintf(`
		SELECT %s
		FROM posts
	`, strings.Join(cols, ","))
	var args []any
	var where []string
	index := 1
	for key, values := range filter {
		if len(values) == 0 {
			continue
		}
		var valueWhere []string
		for _, value := range values {
			valueWhere = append(valueWhere, fmt.Sprintf("$%d", index))
			args = append(args, value)
			index++
		}
		where = append(where, fmt.Sprintf("%s IN (%s)", key, strings.Join(valueWhere, ",")))
	}

	if len(where) > 0 {
		query += fmt.Sprintf(" WHERE %s", strings.Join(where, " AND "))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(
			&post.PostID,
			&post.Title,
			&post.AddResult,
			&post.Body,
			&post.CreatedAt,
			&post.UserID,
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return posts, nil
}
