package errors

import (
	"encoding/json"
	"os"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/models"
)

type KarmaError struct {
	KE []models.ErrorMessage
}

func NewKarmaError() *KarmaError {
	error_definations_path := config.DefaultConfig().ErrorsDefinationFile
	errors, err := os.Open(error_definations_path)
	if err != nil {
		panic(err)
	}
	defer errors.Close()

	decoder := json.NewDecoder(errors)
	var errorDef []models.ErrorMessage
	err = decoder.Decode(&errorDef)
	if err != nil {
		panic(err)
	}
	return &KarmaError{KE: errorDef}
}

func (ke *KarmaError) AddError(errorCode int, description, errorMsg, userMsg string, err_level string) {
	ke.KE = append(ke.KE, models.ErrorMessage{
		ErrorCode:   errorCode,
		Description: description,
		ErrorMsg:    errorMsg,
		UserMsg:     userMsg,
		ErrorLevel:  err_level,
	})
}

func (ke *KarmaError) GetError(errorCode int) models.ErrorMessage {
	for _, error := range ke.KE {
		if error.ErrorCode == errorCode {
			return error
		}
	}
	return models.ErrorMessage{}
}

func (ke *KarmaError) WriteErrorsToFile() error {
	error_definations_path := config.DefaultConfig().ErrorsDefinationFile
	errors, err := os.Create(error_definations_path)
	if err != nil {
		return err
	}
	defer errors.Close()

	encoder := json.NewEncoder(errors)
	err = encoder.Encode(ke.KE)
	if err != nil {
		return err
	}
	return nil
}
