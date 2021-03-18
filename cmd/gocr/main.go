package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/Rhymond/go-money"
	"github.com/jessevdk/go-flags"
	"github.com/montanaflynn/gocr"
	"github.com/theckman/yacspin"

	pdfcpu "github.com/pdfcpu/pdfcpu/pkg/api"
)

var opts struct {
	SkipAgree bool   `long:"agree" description:"Agree to pricing without confirmation."`
	APIKey    string `long:"api-key" description:"Set Mathpix OCR API Key without question."`
}

var inputVal string
var pageCount int

func errorExit(err error) {
	if err != nil {
		fmt.Println(err)
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
	args, err := flags.Parse(&opts)
	if err != nil {
		flagsErr, ok := err.(*flags.Error)
		if ok {
			if flagsErr.Type == flags.ErrHelp {
				os.Exit(0)
			} else {
				os.Exit(1)
			}
		}
		os.Exit(1)
	}

	apiKey := os.Getenv("MATHPIX_OCR_API_KEY")
	if opts.APIKey != "" {
		apiKey = opts.APIKey
	}

	var initialSurvey = []*survey.Question{}
	answers := struct {
		Key    string
		Input  string
		Output string
		Agree  bool
	}{
		Key: apiKey,
	}

	if len(args) >= 1 {
		answers.Input = args[0]
		inputVal = answers.Input
	}
	if len(args) >= 2 {
		answers.Output = args[1]
	}

	if answers.Key == "" {
		initialSurvey = append(initialSurvey, &survey.Question{
			Name: "key",
			Prompt: &survey.Password{
				Message: "What is your mathpix api key?",
			},
			Validate: survey.Required,
		})
	}

	if answers.Input == "" {
		initialSurvey = append(initialSurvey, &survey.Question{
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

	err = survey.Ask(initialSurvey, &answers)
	errorExit(err)

	pageCount, err := pdfcpu.PageCountFile(answers.Input)
	if err != nil {
		errorExit(fmt.Errorf("could not parse %s as a valid pdf", answers.Input))
	}

	costPerPage := money.New(10, "USD")
	totalCost := costPerPage.Multiply(int64(pageCount)).Display()

	var secondSurvey = []*survey.Question{}

	if answers.Output == "" {
		secondSurvey = append(secondSurvey, &survey.Question{
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

	if !opts.SkipAgree {
		secondSurvey = append(secondSurvey, &survey.Question{
			Name: "agree",
			Prompt: &survey.Confirm{
				Message: fmt.Sprintf("Do you agree to the cost of %s for %d pages?", totalCost, pageCount),
			},
			Validate: func(val interface{}) error {
				agree := val.(bool)
				if !agree {
					fmt.Println("You must agree to pricing to continue.")
					os.Exit(1)
					return errors.New("you must agree to pricing to continue")

				}
				return nil
			},
		})
	}

	err = survey.Ask(secondSurvey, &answers)
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
