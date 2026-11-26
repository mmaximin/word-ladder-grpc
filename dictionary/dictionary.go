package dictionary

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"
)

type Dictionary struct {
	Words map[string]bool
}

var instance *Dictionary

var once sync.Once

func GetDictionary() *Dictionary {
	once.Do(func() {
		instance = &Dictionary{
			Words: make(map[string]bool),
		}

		data, err := os.ReadFile("words_dictionary.json")
		if err != nil {
			log.Fatal(err)
		}

		var words map[string]int

		err = json.Unmarshal(data, &words)
		if err != nil {
			log.Fatal(err)
		}

		for word := range words {
			instance.Words[strings.ToUpper(word)] = true
		}
	})

	return instance
}

func (d *Dictionary) Contains(word string) bool {
	return d.Words[strings.ToUpper(word)]
}
