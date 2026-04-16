package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/ashvwinn/unicrm-api/internal/data"
	"github.com/ashvwinn/unicrm-api/internal/validator"
	"github.com/labstack/echo/v4"
)

func (app *application) createClientHandler(c echo.Context) error {
	input := new(data.Client)

	err := echo.FormFieldBinder(c).
		String("client_name", &input.ClientName).
		String("company_name", &input.CompanyName).
		String("email", &input.Email).
		String("phone", &input.Phone).
		String("state", &input.State).
		String("city", &input.City).
		String("segment", &input.Segment).
		BindError()

	if err != nil {
		app.badRequestResponse(c, err)
		return nil
	}

	form, err := c.MultipartForm()
	if err != nil {
		app.badRequestResponse(c, err)
		return nil
	}

	client := data.Client{
		CompanyName: input.CompanyName,
		ClientName:  input.ClientName,
		Email:       input.Email,
		Phone:       input.Phone,
		State:       input.State,
		City:        input.City,
		Segment:     input.Segment,
	}

	v := validator.New()
	if data.ValidateClient(v, &client); !v.Valid() {
		app.failedValidationResponse(c, v.Errors)
		return nil
	}

	err = app.models.Clients.Insert(&client)
	if err != nil {
		app.serverErrorResponse(c, err)
		return nil
	}

	filesMetadata := data.CalculateFilesMetadata(form, client.ID)
	err = data.SaveFilesLocally(form, filesMetadata)
	if err != nil {
		app.serverErrorResponse(c, err)
		return nil
	}

	err = app.models.Files.Insert(filesMetadata)
	if err != nil {
		app.serverErrorResponse(c, err)
		return nil
	}
	client.Files = filesMetadata

	c.Response().Header().Set("Location", fmt.Sprintf("/v1/clients/%d", client.ID))

	return c.JSONPretty(http.StatusCreated, envelope{"client": client}, "\t")
}

func (app *application) showClientHandler(c echo.Context) error {
	id, err := app.readIDParam(c)
	if err != nil {
		app.notFoundResponse(c)
		return nil
	}

	client, err := app.models.Clients.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(c)
		default:
			app.serverErrorResponse(c, err)
		}
		return nil
	}

	files, err := app.models.Files.Get(id)
	if err != nil {
		app.serverErrorResponse(c, err)
		return nil
	}

	client.Files = files

	return c.JSONPretty(http.StatusOK, envelope{"client": client}, "\t")
}

func (app *application) deleteClientHandler(c echo.Context) error {
	id, err := app.readIDParam(c)
	if err != nil {
		app.notFoundResponse(c)
		return nil
	}

	err = os.RemoveAll(fmt.Sprintf("assets/files/%d", id))
	if err != nil {
		app.serverErrorResponse(c, err)
		return nil
	}

	err = app.models.Clients.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(c)
		default:
			app.serverErrorResponse(c, err)
		}
		return nil
	}

	return c.JSONPretty(http.StatusOK, envelope{"message": "client successfully deleted"}, "\t")
}

func (app *application) listClientsHandler(c echo.Context) error {
	var input struct {
		CompanyName string
		State       string
		City        string
		Segment     string
		data.Filters
	}

	v := validator.New()
	qs := c.QueryParams()

	input.CompanyName = app.readString(qs, "company_name", "")
	input.State = app.readString(qs, "state", "")
	input.City = app.readString(qs, "city", "")
	input.Segment = app.readString(qs, "segment", "")

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafelist = []string{
		"id", "company_name", "state", "city", "segment",
		"-id", "-company_name", "-state", "-city", "-segment",
	}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(c, v.Errors)
		return nil
	}

	clients, metadata, err := app.models.Clients.GetAll(
		input.CompanyName,
		input.State,
		input.City,
		input.Segment,
		input.Filters,
	)

	if err != nil {
		app.serverErrorResponse(c, err)
		return nil
	}

	return c.JSONPretty(http.StatusOK, envelope{"clients": clients, "metadata": metadata}, "\t")
}

func (app *application) updateClientHandler(c echo.Context) error {
	id, err := app.readIDParam(c)
	if err != nil {
		app.notFoundResponse(c)
		return nil
	}

	var input struct {
		CompanyName string `json:"company_name"`
		ClientName  string `json:"client_name"`
		Email       string `json:"email"`
		Phone       string `json:"phone"`
		State       string `json:"state"`
		City        string `json:"city"`
		Segment     string `json:"segment"`
	}

	err = c.Bind(&input)
	if err != nil {
		app.badRequestResponse(c, err)
		return nil
	}

	client := new(data.Client)
	client.ID = id
	client.CompanyName = input.CompanyName
	client.ClientName = input.ClientName
	client.Email = input.Email
	client.Phone = input.Phone
	client.State = input.State
	client.City = input.City
	client.Segment = input.Segment

	v := validator.New()
	if data.ValidateClient(v, client); !v.Valid() {
		app.failedValidationResponse(c, v.Errors)
		return nil
	}

	err = app.models.Clients.Update(client)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(c)
		default:
			app.serverErrorResponse(c, err)
		}
		return nil
	}

	return c.JSONPretty(http.StatusOK, envelope{"client": client}, "\t")
}
