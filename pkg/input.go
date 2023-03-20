package gltr

import (
	"fmt"
	"os"

	"github.com/erikgeiser/promptkit/confirmation"
	"github.com/erikgeiser/promptkit/selection"
	"github.com/erikgeiser/promptkit/textinput"
)

func ReadTextInput(prompt, defaultValue, placeholder string) string {
	input := textinput.New(prompt)
	input.InitialValue = defaultValue
	input.Placeholder = placeholder

	name, err := input.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		os.Exit(1)
	}

	return name
}

func ReadOptionInput(prompt, defaultValue string, options []string) string {
	input := selection.New(prompt, options)
	input.Filter = nil
	// input.Filter = false

	name, err := input.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		os.Exit(1)
	}

	return name
}

// trye corresponds to yes, false to no...
func ReadConfirmationInput(prompt string, defaultValue confirmation.Value) bool {
	// input.Filter = false
	var initialValue confirmation.Value
	input := confirmation.New(prompt, initialValue)

	confirmation, err := input.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		os.Exit(1)
	}

	return confirmation
}
