package tests

import (
	"fmt"

	"github.com/MelloB1989/karma/ai"
)

func TestImageGen() {
	kimg := ai.NewKarmaImageGen(ai.SEGMIND_SD, ai.WithNImages(1), ai.WithOutputDirectory("./images"))
	url, err := kimg.GenerateImages("A cute robot in a forest of trees.")
	if err != nil {
		panic(err)
	}
	fmt.Println(url)
}

func TestNanoBananaImageGen() {
	kimg := ai.NewKarmaImageGen(ai.SEGMIND_NANO_BANANA, ai.WithNImages(1), ai.WithOutputDirectory("./images"))

	// Nano Banana requires input images
	inputImages := []string{
		"https://segmind-resources.s3.amazonaws.com/input/09a99645-3171-4742-be08-dfcfe7f0a4b2-1304f734-929b-4047-822d-4f59fca2179a-40457f0b-d422-4525-b3a5-19633a9cdac0.png",
	}

	url, err := kimg.GenerateImagesWithInputImages("Dancing Banana", inputImages)
	if err != nil {
		panic(err)
	}
	fmt.Println("Nano Banana result:", url)
}

func TestGeminiImageGen() {
	// // With SpecialConfig overrides
	gen := ai.NewKarmaImageGen(ai.GEMINI_3_PRO_IMAGE,
		ai.WithImgSpecialConfig(map[ai.SpecialConfig]any{
			ai.GoogleLocation: "global",
		}),
		ai.WithImgAspectRatio("16:9"),
		ai.WithImgDisabledSafetyFilters(),
	)
	response, err := gen.GenerateImages("A futuristic Lamborghini")
	if err != nil {
		panic(err)
	}
	fmt.Println("Gemini result:", response)

}
