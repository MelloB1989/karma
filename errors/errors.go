package errors

import (
	"encoding/json"
	"os"

	"github.com/MelloB1989/karma/config"
)

type ErrorMessage struct {
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
	ErrorMsg    string `json:"error_msg"`
	UserMsg     string `json:"user_msg"`
}

type KarmaError struct {
	KE []ErrorMessage
}

func NewKarmaError() *KarmaError {
	error_definations_path := config.DefaultConfig().ErrorsDefinationFile
	errors, err := os.Open(error_definations_path)
	if err != nil {
		panic(err)
	}
	defer errors.Close()

	decoder := json.NewDecoder(errors)
	var errorDef []ErrorMessage
	err = decoder.Decode(&errorDef)
	if err != nil {
		panic(err)
	}
	return &KarmaError{KE: errorDef}
}

func (ke *KarmaError) AddError(errorCode int, description, errorMsg, userMsg string) {
	ke.KE = append(ke.KE, ErrorMessage{
		ErrorCode:   errorCode,
		Description: description,
		ErrorMsg:    errorMsg,
		UserMsg:     userMsg,
	})
}

func (ke *KarmaError) GetError(errorCode int) ErrorMessage {
	for _, error := range ke.KE {
		if error.ErrorCode == errorCode {
			return error
		}
	}
	return ErrorMessage{}
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
