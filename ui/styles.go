package ui

import (
	"log"
	"os"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func ApplyStyles() {
	cssPath := "assets/css/style.css"
	if _, err := os.Stat(cssPath); os.IsNotExist(err) {
		cssPath = "/app/share/circadia/assets/css/style.css"
	}

	provider := gtk.NewCSSProvider()
	provider.LoadFromPath(cssPath)

	display := gdk.DisplayGetDefault()
	if display == nil {
		log.Println("Could not get default display, skipping stylesheet")
		return
	}

	gtk.StyleContextAddProviderForDisplay(display, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
