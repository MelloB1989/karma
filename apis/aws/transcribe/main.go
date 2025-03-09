package transcribe

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	c "github.com/MelloB1989/karma/config"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming"
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming/types"
	"github.com/aws/aws-sdk-go/aws"
)

const (
	LanguageCodeEnUS = "en-US"
	LanguageCodeEsUS = "es-US"
	LanguageCodeFrCA = "fr-CA"
	LanguageCodeEnGB = "en-GB"
	LanguageCodeEnAU = "en-AU"
	LanguageCodeFrFR = "fr-FR"
	LanguageCodeItIT = "it-IT"
	LanguageCodeDeDE = "de-DE"
	LanguageCodePtBR = "pt-BR"
	LanguageCodeJaJP = "ja-JP"
	LanguageCodeKoKR = "ko-KR"
	LanguageCodeZhCN = "zh-CN"
)

// Media encoding formats supported by AWS Transcribe
const (
	MediaEncodingPCM  = "pcm"
	MediaEncodingOGG  = "ogg-opus"
	MediaEncodingFLAC = "flac"
	MediaEncodingMP3  = "mp3"
	MediaEncodingMP4  = "mp4"
	MediaEncodingAMR  = "amr"
	MediaEncodingWEBM = "webm"
)

func NewTranscribeClient() (*transcribestreaming.Client, error) {
	// Create AWS SDK configuration
	sdkOpts := []func(*config.LoadOptions) error{
		config.WithRetryMaxAttempts(3),
	}

	r, err := c.GetEnv("AWS_TRANSCRIBE_REGION")
	if r != "" {
		sdkOpts = append(sdkOpts, config.WithRegion(r))
	} else {
		// Default region if not specified
		sdkOpts = append(sdkOpts, config.WithRegion("ap-south-1"))
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(), sdkOpts...)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	// Create Transcribe client
	client := transcribestreaming.NewFromConfig(awsCfg)

	return client, nil
}

func StartStream() {
	filePath := "/Users/mellob/Downloads/stream.opus"
	audio, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer audio.Close()
	client, err := NewTranscribeClient()
	if err != nil {
		log.Fatalf("failed to create transcribe client: %v", err)
	}

	resp, err := client.StartStreamTranscription(context.Background(), &transcribestreaming.StartStreamTranscriptionInput{
		LanguageCode:         types.LanguageCode("en-US"),
		MediaEncoding:        types.MediaEncoding("ogg-opus"),
		MediaSampleRateHertz: aws.Int32(44100),
	})
	if err != nil {
		log.Fatalf("failed to start streaming, %v", err)
	}
	stream := resp.GetStream()
	defer stream.Close()

	// Check if audio is nil
	if audio == nil {
		log.Fatalf("audio source is nil")
	}

	// Start streaming audio in a goroutine
	go StreamAudioFromReader(context.Background(), stream.Writer, 10*1024, audio)

	fmt.Println(stream.Events())
	// Process the transcription events
	for event := range stream.Events() {
		switch e := event.(type) {
		case *types.TranscriptEvent:
			log.Printf("got event, %v results", len(e.Transcript.Results))
			for _, res := range e.Transcript.Results {
				for _, alt := range res.Alternatives {
					log.Printf("* %s", aws.StringValue(alt.Transcript))
				}
			}
		default:
			log.Fatalf("unexpected event, %T", event)
		}
	}
	// for event := range stream.Events() {
	// 	// Just use type reflection to log the event type and handle generic access
	// 	eventType := fmt.Sprintf("%T", event)
	// 	log.Printf("Got event of type: %s", eventType)

	// 	// Use reflection to extract transcript information from any struct
	// 	v := reflect.ValueOf(event)
	// 	if v.Kind() == reflect.Ptr {
	// 		v = v.Elem()
	// 	}

	// 	// Try to navigate the structure to find transcript results
	// 	if v.Kind() == reflect.Struct {
	// 		// Look for TranscriptEvent or Transcript field
	// 		for i := 0; i < v.NumField(); i++ {
	// 			fieldName := v.Type().Field(i).Name
	// 			if fieldName == "Transcript" {
	// 				transcript := v.Field(i)
	// 				if transcript.Kind() == reflect.Struct {
	// 					// Look for Results field
	// 					for j := 0; j < transcript.NumField(); j++ {
	// 						resultsFieldName := transcript.Type().Field(j).Name
	// 						if resultsFieldName == "Results" {
	// 							results := transcript.Field(j)
	// 							if results.Kind() == reflect.Slice {
	// 								log.Printf("Found %d results", results.Len())
	// 								// Process each result
	// 								for k := 0; k < results.Len(); k++ {
	// 									result := results.Index(k)
	// 									if result.Kind() == reflect.Struct {
	// 										// Look for Alternatives field
	// 										for l := 0; l < result.NumField(); l++ {
	// 											altFieldName := result.Type().Field(l).Name
	// 											if altFieldName == "Alternatives" {
	// 												alts := result.Field(l)
	// 												if alts.Kind() == reflect.Slice {
	// 													for m := 0; m < alts.Len(); m++ {
	// 														alt := alts.Index(m)
	// 														if alt.Kind() == reflect.Struct {
	// 															for n := 0; n < alt.NumField(); n++ {
	// 																transcriptFieldName := alt.Type().Field(n).Name
	// 																if transcriptFieldName == "Transcript" {
	// 																	transcriptField := alt.Field(n)
	// 																	if transcriptField.Kind() == reflect.Ptr && !transcriptField.IsNil() {
	// 																		log.Printf("* %s", transcriptField.Elem().String())
	// 																	}
	// 																}
	// 															}
	// 														}
	// 													}
	// 												}
	// 											}
	// 										}
	// 									}
	// 								}
	// 							}
	// 						}
	// 					}
	// 				}
	// 			}
	// 		}
	// 	}
	// }

	if err := stream.Err(); err != nil {
		log.Fatalf("expect no error from stream, got %v", err)
	}
}

// StreamAudioFromReader reads audio data from the provided reader and streams it to AWS Transcribe
func StreamAudioFromReader(ctx context.Context, writer transcribestreaming.AudioStreamWriter, chunkSize int, audio io.Reader) error {
	buffer := make([]byte, chunkSize)

	for {
		// Check if context is done
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue processing
		}

		// Read a chunk of audio data
		n, err := audio.Read(buffer)
		if err != nil {
			if err == io.EOF {
				// End of file, close the stream
				if closeErr := writer.Close(); closeErr != nil {
					return fmt.Errorf("error closing audio stream: %w", closeErr)
				}
				return nil
			}
			return fmt.Errorf("error reading audio data: %w", err)
		}

		// If we read some data, send it
		if n > 0 {
			// Make a copy of the data to avoid buffer reuse issues
			chunk := make([]byte, n)
			copy(chunk, buffer[:n])

			// For AWS SDK Go v2 v1.36.3, we need to determine the specific type
			// Let's import the necessary code to access the exact type needed

			// Try with AudioEvent
			audioEvent := &types.AudioEvent{
				AudioChunk: chunk,
			}

			// Use type assertion to convert to AudioStream
			// or create a struct that satisfies the interface

			// Let's first try an attempt with the raw struct
			if err := writer.Send(ctx, struct {
				types.AudioStream
				*types.AudioEvent
			}{
				AudioEvent: audioEvent,
			}); err != nil {
				return fmt.Errorf("error writing audio to stream: %w", err)
			}
		}
	}
}
