package main

import (
	"log"
	"math/rand"
        "os"
	"time"

	"gopkg.in/qml.v1"
)

func main() {

	err := qml.Run(run)
	if err != nil {
		log.Panic("Could not run QML man", err)
	}
}

func run() error {
	engine := qml.NewEngine()

	component, err := engine.LoadFile("hello-world.qml")
	if err != nil {
		return err
	}

        dir, _ := os.Getwd()
	ctrl := Control{Message: dir}

	context := engine.Context()
	context.SetVar("ctrl", &ctrl)

	window := component.CreateWindow(nil)

	ctrl.Root = window.Root()

	rand.Seed(time.Now().Unix())

	window.Show()
	time.Sleep(5000 * time.Millisecond)

	return nil

}

type Control struct {
	Root    qml.Object
	Message string
}
