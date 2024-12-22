package tests

import (
	"fmt"

	"github.com/MelloB1989/karma/errors"
)

func TestKarmaErrorPackage() {
	ke := errors.NewKarmaError()
	fmt.Println(ke.GetError(1))
}
