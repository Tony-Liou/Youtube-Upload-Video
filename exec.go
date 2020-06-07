package main

import (
	"bytes"
	"fmt"
	"os/exec"
)

func main() {
	app := "streamlink"

	arg0 := "-o"
	arg1 := "filename.flv"
	arg2 := "https://17.live/live/3744274"
	arg3 := "best"

	cmd := exec.Command(app, arg0, arg1, arg2, arg3)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return
	}
	fmt.Println("Result: " + out.String())
}
