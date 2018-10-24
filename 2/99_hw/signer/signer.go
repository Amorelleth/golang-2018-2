package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

const th = 6

func CalculateCrc32Hash(val string, ch chan string) {
	res := DataSignerCrc32(val)
	ch <- res
}

func CalculateSingleHash(val string, vMd5 string, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()

	ch1 := make(chan string)
	ch2 := make(chan string)

	go CalculateCrc32Hash(val, ch1)
	go CalculateCrc32Hash(vMd5, ch2)

	hash1 := <-ch1
	hash2 := <-ch2

	out <- hash1 + "~" + hash2
}

func CalculateMultiHashByIdx(v string, hashArr []string, idx int, wg *sync.WaitGroup) {
	defer wg.Done()

	hash := DataSignerCrc32(v)
	hashArr[idx] = hash
}

func CalculateMultiHash(val interface{}, out chan interface{}, wg *sync.WaitGroup) {
	res := make([]string, th)
	wgInner := &sync.WaitGroup{}

	defer wg.Done()

	for i := 0; i < th; i++ {
		wgInner.Add(1)
		v := strconv.Itoa(i) + val.(string)

		go CalculateMultiHashByIdx(v, res, i, wgInner)
	}

	wgInner.Wait()
	out <- strings.Join(res, "")
}

func SingleHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}

	for val := range in {
		wg.Add(1)

		v := strconv.Itoa(val.(int))
		vMd5 := DataSignerMd5(v)

		go CalculateSingleHash(v, vMd5, out, wg)
	}

	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}

	for val := range in {
		wg.Add(1)
		go CalculateMultiHash(val, out, wg)
	}

	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	arr := []string{}

	for val := range in {
		arr = append(arr, val.(string))
	}

	sort.Strings(arr)

	out <- strings.Join(arr, "_")
}

func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}
	in := make(chan interface{}, 100)

	defer wg.Wait()

	for _, j := range jobs {
		wg.Add(1)

		out := make(chan interface{}, 100)

		go func(j job, inCh, newOut chan interface{}) {
			defer wg.Done()
			defer close(out)

			j(inCh, newOut)
		}(j, in, out)

		in = out
	}
}
