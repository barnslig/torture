package main

import (
	"reflect"
)

type Filter struct {
	SmallFiles bool
	LargeFiles bool
}

func CreateFilter() (filter *Filter) {
	filter = &Filter{}

	return
}

// Tells if any filter is set
// Currently boilerplate. To be used later, maybe!
func (filter *Filter) IsFiltered() bool {
	v := reflect.ValueOf(*filter)

	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Interface() == true {
			return true
		}
	}

	return false
}

// Parses a slice of strings, e.g. []string{"small", "nsfw"} into a Filter struct
func (filter *Filter) UnmarshalStringSlice(input []string) {
	for _, el := range input {
		switch el {
		case "small":
			filter.SmallFiles = true
			break
		case "large":
			filter.LargeFiles = true
			break
		}
	}
}
