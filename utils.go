package main

import (
	"log"
	"math/rand"
	"time"
	"unsafe"
)

func genRoomNum() string {
	n := roomIDlen
	b := make([]byte, n)
	var src = rand.NewSource(time.Now().UnixNano())
	const letterBytes = "abcdefghijkmnopqrstuvwxyz023456789"
	const (
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}

func randInt(min, max int) int {
	rand.Seed(time.Now().Unix())
	return min + rand.Intn(max-min+1)
}

func deDupe(slice []int) (list []int) {
	keys := make(map[int]bool)
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return
}

func remove(s []int, i int) (o []int) {
	for _, e := range s {
		if e != i {
			o = append(o, e)
		}
	}
	return
}

func check(e error) {
	if e != nil {
		log.Panic(e)
	}
}
