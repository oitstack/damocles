package main

import (
	"fmt"
	sword "goblin_watchdog/src/sword"
	"math/big"
	"os"
)

func main() {

	fmt.Println("starting the Sword.")
	sword := sword.NewTheSword(os.Getenv("targetName"), *big.NewInt(20))
	sword.Start()
	fmt.Println("the Sword has kill some men.")
}
