package main

import (
	"fmt"
	"strconv"

	"github.com/karrick/congomap"
)

func main() {
	var series congomap.Congomap

	lookup := func(key string) (interface{}, error) {
		value, err := strconv.Atoi(key)
		if err != nil {
			return nil, err
		}
		if value < 2 {
			return 1, nil
		}
		first, err := series.LoadStore(strconv.Itoa(value - 1))
		if err != nil {
			return nil, err
		}
		second, err := series.LoadStore(strconv.Itoa(value - 2))
		if err != nil {
			return nil, err
		}
		return first.(int) + second.(int), nil
	}

	config := &congomap.Config{Lookup: lookup}

	var err error
	series, err = congomap.NewTwoLevelMap(config)
	if err != nil {
		panic(err)
	}

	_, err = series.LoadStore("10")
	if err != nil {
		panic(err)
	}

	for pair := range series.Pairs() {
		fmt.Println(pair)
	}
}
