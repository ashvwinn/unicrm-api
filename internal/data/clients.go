package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ashvwinn/unicrm-api/internal/validator"
)

type Client struct {
	ID          int       `json:"id"`
	CreatedAt   time.Time `json:"-"`
	CompanyName string    `json:"company_name"`
	ClientName  string    `json:"client_name"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	State       string    `json:"state"`
	City        string    `json:"city"`
	Segment     string    `json:"segment"`
	Files       []File    `json:"files"`
}

type ClientModel struct {
	DB *sql.DB
}

func ValidateClient(v *validator.Validator, client *Client) {
	v.Check(client.CompanyName != "", "company_name", "must be provided")
	v.Check(len(client.CompanyName) <= 100, "company_name", "must not be more than 100 bytes long")

	v.Check(client.ClientName != "", "client_name", "must be provided")
	v.Check(len(client.ClientName) <= 100, "client_name", "must not be more than 100 bytes long")

	v.Check(client.State != "", "state", "must be provided")
	v.Check(len(client.State) <= 50, "state", "must not be more than 50 bytes long")
	v.Check(client.City != "", "city", "must be provided")
	v.Check(len(client.City) <= 50, "city", "must not be more than 50 bytes long")

	v.Check(validator.Matches(client.Email, validator.EmailRX), "email", "must be a valid email address")
	v.Check(len(client.Phone) == 10, "phone", "must be 10 bytes long")
}

func (m ClientModel) Insert(client *Client) error {
	query := `
        INSERT INTO clients (company_name, client_name, email, phone, state, city, segment)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id, created_at`

	args := []any{
		client.CompanyName,
		client.ClientName,
		client.Email,
		client.Phone,
		client.State,
		client.City,
		client.Segment,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&client.ID, &client.CreatedAt)
}

func (m ClientModel) Get(id int) (*Client, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
        SELECT id, created_at, company_name, client_name, email, phone, state, city, segment
        FROM clients
        WHERE id = $1`

	var client Client

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&client.ID,
		&client.CreatedAt,
		&client.CompanyName,
		&client.ClientName,
		&client.Email,
		&client.Phone,
		&client.State,
		&client.City,
		&client.Segment,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &client, nil
}

func (m ClientModel) Delete(id int) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	query := `
        DELETE FROM clients
        WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (m ClientModel) Update(client *Client) error {
	query := `
        UPDATE clients
        SET company_name = $1, client_name = $2, email = $3, phone = $4, state = $5, city = $6, segment = $7
        WHERE id = $8`

	args := []any{
		client.CompanyName,
		client.ClientName,
		client.Email,
		client.Phone,
		client.State,
		client.City,
		client.Segment,
		client.ID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := m.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	if rowsAffected, err := res.RowsAffected(); err != nil {
		return err
	} else if rowsAffected == 0 {
		return ErrEditConflict
	}

	return nil
}

func (m ClientModel) GetAll(companyName, state, city, segment string, filters Filters) ([]*Client, Metadata, error) {
	query := fmt.Sprintf(`
        SELECT count(*) OVER(), id, created_at, company_name, client_name, email, phone, state, city, segment
        FROM clients
        WHERE (SIMILARITY(company_name, $1) > 0 OR $1 = '')
        AND (LOWER(state) = LOWER($2) OR $2 = '')
        AND (LOWER(city) = LOWER($3) OR $3 = '')
        AND (LOWER(segment) = LOWER($4) OR $4 = '')
        ORDER BY %s %s, id
        LIMIT $5 OFFSET $6`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{companyName, state, city, segment, filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	clients := []*Client{}
	for rows.Next() {
		var client Client

		err := rows.Scan(
			&totalRecords,
			&client.ID,
			&client.CreatedAt,
			&client.CompanyName,
			&client.ClientName,
			&client.Email,
			&client.Phone,
			&client.State,
			&client.City,
			&client.Segment,
		)

		if err != nil {
			return nil, Metadata{}, err
		}

		clients = append(clients, &client)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return clients, metadata, nil
}
