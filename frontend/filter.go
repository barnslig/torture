package main

import (
	"reflect"
)

type Filter struct {
	SmallFiles bool
}

func CreateFilter() (filter *Filter) {
	filter = &Filter{}

	return
}

func (filter *Filter) IsUnfiltered() bool {
	v := reflect.ValueOf(*filter)

	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Interface() == true {
			return true
		}
	}

	return false
}

func (filter *Filter) UnmarshalStringSlice(input []string) {
	for _, el := range input {
		switch el {
		case "small":
			filter.SmallFiles = true
		}
	}
}
