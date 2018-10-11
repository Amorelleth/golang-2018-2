package main

import (
	"sort"
	"strconv"
)

func IntSliceToString(slice []int) string {
	var str string

	for _, val := range slice {
		str += strconv.Itoa(val)
	}

	return str
}

func MergeSlices(sl1 []float32, sl2 []int32) []int {
	var mergedSlice []int

	for _, val := range sl1 {
		mergedSlice = append(mergedSlice, int(val))
	}

	for _, val := range sl2 {
		mergedSlice = append(mergedSlice, int(val))
	}

	return mergedSlice
}

func GetMapValuesSortedByKey(m map[int]string) []string {
	var keys []int
	var res []string

	for k := range m {
		keys = append(keys, k)
	}

	sort.Ints(keys)

	for _, k := range keys {
		res = append(res, m[k])
	}

	return res
}
