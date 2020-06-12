package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("invalid arguments")
	}

	num, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("timestamp,yhat,yhat_upper,yhat_lower")
	timestamp := time.Now().Unix()
	rand.Seed(time.Now().Unix())
	for i := 0; i < num; i++ {
		yhatLower := rand.Float64() * 100.0
		yhat := rand.Float64()*100.0 + 100
		yhatUpper := rand.Float64()*100.0 + 200
		timestamp += 60
		fmt.Printf("%d,%.2f,%.2f,%.2f\n", timestamp, yhat, yhatUpper, yhatLower)
	}
}
