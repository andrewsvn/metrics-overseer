package pkg1

import (
	"log"
	"os"
	"time"
)

func main() {
	x := time.Now().UnixNano()
	if x%10 == 0 {
		os.Exit(0) // want "os.Exit should not be called outside main function"
	}
	if x%9 == 0 {
		log.Fatal("") // want "log.Fatal should not be called outside main function"
	}
	if x%8 == 0 {
		log.Fatalf("%s", "") // want "log.Fatalf should not be called outside main function"
	}
	if x%7 == 0 {
		panic(nil) // want "panic should not be used"
	}
}
