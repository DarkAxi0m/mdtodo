package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

// KeyBindings holds the key mapping
type KeyBindings struct {
	Quit     string `json:"Quit"`
	Load     string `json:"Load"`
	Save     string `json:"Save"`
	ShowDone string `json:"ShowDone"`

	ShowNotes string `json:"SnowNotes"`
	EditNotes string `json:"EditNotes"`

	MoveUp    string `json:"MoveUp"`
	MoveDown  string `json:"MoveDown"`
	ShiftUp   string `json:"ShiftUp"`
	ShiftDown string `json:"ShiftDown"`

	Delete string `json:"Delete"`

	AddTask    string `json:"AddTask"`
	EditTask   string `json:"EditTask"`
	TagTask    string `json:"TagTask"`
	ToggleTask string `json:"ToggleTask"`

	ModeProject string `json:"ModeProject"`
	ModeTask    string `json:"ModeTask"`
}

// Applies non-zero fields from src to dest
func mergeNonEmptyFields(dest, src interface{}) {
	destVal := reflect.ValueOf(dest).Elem()
	srcVal := reflect.ValueOf(src).Elem()

	for i := 0; i < destVal.NumField(); i++ {
		field := srcVal.Field(i)
		if !field.IsZero() {
			destVal.Field(i).Set(field)
		}
	}
}

func defaultKeyBindings() *KeyBindings {
	return &KeyBindings{
		Quit:     "q",
		Load:     "l",
		Save:     "w",
		ShowDone: "h",

		ShowNotes: "N",
		EditNotes: "n",

		Delete: "d",

		MoveUp:    "k",
		MoveDown:  "j",
		ShiftUp:   "K",
		ShiftDown: "J",

		AddTask:    "i",
		EditTask:   "I",
		TagTask:    "e",
		ToggleTask: " ",

		ModeProject: "p",
		ModeTask:    "t",
	}
}

func LoadKeyBindingsWithDefaults(filename string) *KeyBindings {
	defaults := defaultKeyBindings()

	file, err := os.Open(filename)
	if err != nil {
		return defaults
	}
	defer file.Close()

	var loaded KeyBindings
	if err := json.NewDecoder(file).Decode(&loaded); err != nil {
		fmt.Println("Error decoding keybindings file:", err)
		return defaults
	}

	mergeNonEmptyFields(defaults, &loaded)
	return defaults
}

func saveKeyBindings(filename string, bindings *KeyBindings) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty-print JSON
	return encoder.Encode(bindings)
}

func LoadKeyBindings() *KeyBindings {
	path, err := getUserConfigPath(BindingConfig)
	if err != nil {
		panic(err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		_ = os.MkdirAll(filepath.Dir(path), 0755)
		_ = saveKeyBindings(path, defaultKeyBindings())
	}

	bindings := LoadKeyBindingsWithDefaults(path)

	return bindings
}
