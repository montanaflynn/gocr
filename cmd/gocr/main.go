package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/montanaflynn/gocr"
	"github.com/theckman/yacspin"
)

var inputVal string

func errorExit(err error) {
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
}

func suggestFiles(toComplete string) []string {
	files, _ := filepath.Glob(toComplete + "*.pdf")
	return files
}

func suggestOutput(toComplete string) []string {
	return []string{inputVal + ".mmd", inputVal + ".docx", inputVal + ".tex.zip"}
}

func validateExtension(path string, validExtensions []string) error {
	var pathExtension = filepath.Ext(path)
	for _, validExtension := range validExtensions {
		if pathExtension == validExtension {
			return nil
		}

	}
	return fmt.Errorf("%s does not end with one of: %s", path, validExtensions)
}

func main() {

	apiKey := os.Getenv("MATHPIX_OCR_API_KEY")
	args := os.Args[1:]

	var prompt = []*survey.Question{}
	answers := struct {
		Key    string
		Input  string
		Output string
	}{
		Key: apiKey,
	}

	if len(args) >= 1 {
		answers.Input = args[0]
	}
	if len(args) >= 2 {
		answers.Output = args[1]
	}

	if answers.Key == "" {
		prompt = append(prompt, &survey.Question{
			Name: "key",
			Prompt: &survey.Input{
				Message: "What is your api key?",
			},
			Validate: survey.Required,
		})
	}

	if answers.Input == "" {
		prompt = append(prompt, &survey.Question{
			Name: "input",
			Prompt: &survey.Input{
				Message: "Which file should be used as an input?",
				Suggest: suggestFiles,
				Help:    "Select the PDF File to use as input.",
			},
			Validate: func(val interface{}) error {
				input := val.(string)
				if input == "" {
					return errors.New("input is required")
				}

				err := validateExtension(input, []string{".pdf"})
				if err != nil {
					return err
				}

				inputVal = input[:len(input)-4]
				return nil
			},
		})
	}

	if answers.Output == "" {
		prompt = append(prompt, &survey.Question{
			Name: "output",
			Prompt: &survey.Input{
				Message: "Where would you like to save it?",
				Suggest: suggestOutput,
				Help:    "Path to where to save the output format.",
			},
			Validate: func(val interface{}) error {
				output := val.(string)
				if output == "" {
					return errors.New("output is required")
				}

				err := validateExtension(output, []string{".mmd", ".docx", ".tex", ".zip"})
				if err != nil {
					return err
				}

				return nil
			},
		})
	}

	err := survey.Ask(prompt, &answers)
	errorExit(err)

	client := gocr.NewClient(answers.Key)

	cfg := yacspin.Config{
		Frequency:     100 * time.Millisecond,
		CharSet:       yacspin.CharSets[14],
		Message:       " Uploading",
		StopMessage:   fmt.Sprintf(" Saved %s", answers.Output),
		StopCharacter: "✓",
		HideCursor:    true,
	}

	spinner, err := yacspin.New(cfg)
	errorExit(err)

	client.SetSpinner(spinner)

	spinner.Start()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		spinner.StopFail()
		os.Exit(1)
	}()

	err = client.Convert(answers.Input, answers.Output)
	if err != nil {
		spinner.StopFail()
		errorExit(err)
	}

	spinner.Stop()
}
