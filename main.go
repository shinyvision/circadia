package main

import (
	"io"
	"log"
	"os"

	"circadia/daemon"
	"circadia/storage"
	"circadia/ui"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func main() {
	debugMode := false
	filteredArgs := []string{}
	for _, arg := range os.Args {
		if arg == "--debug" {
			debugMode = true
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	if !debugMode {
		log.SetOutput(io.Discard)
	} else {
		log.Println("Debug mode enabled")
	}

	app := gtk.NewApplication("io.github.shinyvision.Circadia", gio.ApplicationFlagsNone)

	app.ConnectStartup(func() {
		log.Println("Service Started")

		if err := storage.InitDB(""); err != nil {
			log.Printf("Failed to init DB: %v", err)
			return
		}

		daemon.SetDebugMode(debugMode)
		daemon.Start(&app.Application)

		log.Println("Holding application for daemon persistence")
		app.Hold()
	})

	var mainWindow *gtk.ApplicationWindow

	app.ConnectActivate(func() {
		log.Println("Activation requested - Opening Window")

		if mainWindow != nil {
			log.Println("Window already open, presenting it")
			mainWindow.Present()
			return
		}

		ui.ApplyStyles()
		mainWindow = NewWindow(app, debugMode)

		mainWindow.ConnectDestroy(func() {
			log.Println("Window destroyed")
			mainWindow = nil
		})

		mainWindow.SetVisible(true)
	})

	if code := app.Run(filteredArgs); code > 0 {
		log.Printf("Application exited with failure code: %d", code)
		os.Exit(code)
	}
}
