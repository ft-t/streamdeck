.PHONY: build-apimonkey-windows
build-apimonkey-windows:
	@cd cmd/client/apimonkey && rm -rf dist && GOOS=windows go build -o dist/com.ftt.apimonkey.exe

.PHONY: dev-apimonkey
dev-apimonkey: build-apimonkey-windows
	@cp cmd/client/apimonkey/dist/com.ftt.apimonkey.exe  /mnt/c/Users/iqpir/AppData/Roaming/Elgato/StreamDeck/Plugins/com.ftt.apimonkey.sdPlugin/ -f
	@rm -rf /mnt/c/Users/iqpir/AppData/Roaming/Elgato/StreamDeck/Plugins/com.ftt.apimonkey.sdPlugin/logs
