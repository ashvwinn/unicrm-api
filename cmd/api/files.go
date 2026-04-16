package main

import (
	"errors"
	"net/http"

	"github.com/ashvwinn/unicrm-api/internal/data"
	"github.com/labstack/echo/v4"
)

func (app *application) createFileHandler(c echo.Context) error {
	clientId, err := app.readIDParam(c)
	if err != nil {
		app.notFoundResponse(c)
		return nil
	}

	file := new(data.File)
	file.Category = c.FormValue("category")
	file.ClientID = clientId

	fileHeader, err := c.FormFile("file")
	if err != nil || fileHeader == nil {
		app.badRequestResponse(c, err)
		return nil
	}

	err = data.SaveFileLocally(fileHeader, file)
	if err != nil {
		app.serverErrorResponse(c, err)
		return nil
	}

	files := []data.File{*file}
	err = app.models.Files.Insert(files)
	if err != nil {
		app.serverErrorResponse(c, err)
		return nil
	}

	return c.JSONPretty(http.StatusCreated, envelope{"file": files[0]}, "\t")
}

func (app *application) deleteFileHandler(c echo.Context) error {
	fileId, err := app.readIDParam(c)
	if err != nil {
		app.notFoundResponse(c)
		return nil
	}

	clientId, err := app.readIntParam(c, "clientId")
	if err != nil {
		app.notFoundResponse(c)
		return nil
	}

	filepath, err := app.models.Files.Delete(fileId, clientId)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(c)
		default:
			app.serverErrorResponse(c, err)
		}
		return nil
	}

	err = data.DeleteFileLocally(filepath)
	if err != nil {
		app.serverErrorResponse(c, err)
		return nil
	}

	return c.JSONPretty(http.StatusOK, envelope{"message": "file successfully deleted"}, "\t")
}
