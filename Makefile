.PHONY: all install

install:
	@if [ ! -f circadia-bin ]; then \
		go build -o circadia-bin .; \
	else \
		echo "Using existing binary: circadia-bin"; \
	fi
	@echo "Installing files..."
	install -D -m 755 circadia-bin /app/bin/circadia
	install -D -m 644 io.github.shinyvision.Circadia.desktop /app/share/applications/io.github.shinyvision.Circadia.desktop
	install -D -m 644 io.github.shinyvision.Circadia.autostart.desktop /app/etc/xdg/autostart/io.github.shinyvision.Circadia.autostart.desktop
	install -D -m 644 io.github.shinyvision.Circadia.metainfo.xml /app/share/metainfo/io.github.shinyvision.Circadia.metainfo.xml
	mkdir -p /app/share/circadia
	cp -r assets /app/share/circadia/
	install -D -m 644 assets/appicon.png /app/share/icons/hicolor/512x512/apps/io.github.shinyvision.Circadia.png
	install -D -m 644 assets/icons/16x16.png /app/share/icons/hicolor/16x16/apps/io.github.shinyvision.Circadia.png
	install -D -m 644 assets/icons/32x32.png /app/share/icons/hicolor/32x32/apps/io.github.shinyvision.Circadia.png
	install -D -m 644 assets/icons/64x64.png /app/share/icons/hicolor/64x64/apps/io.github.shinyvision.Circadia.png
	install -D -m 644 assets/icons/128x128.png /app/share/icons/hicolor/128x128/apps/io.github.shinyvision.Circadia.png
	install -D -m 644 assets/icons/256x256.png /app/share/icons/hicolor/256x256/apps/io.github.shinyvision.Circadia.png
