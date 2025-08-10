package tests

import (
	"fmt"

	"github.com/MelloB1989/karma/ai"
)

func TestImageGen() {
	kimg := ai.NewKarmaImageGen(ai.DALL_E_2, ai.WithNImages(1))
	url, err := kimg.GenerateImages("A cute robot in a forest of trees.")
	if err != nil {
		panic(err)
	}
	fmt.Println(url)
}
