DIST := dist

.PHONY: build clean install uninstall

build:
	mkdir -p $(DIST)
	go build -o $(DIST)/aegisctl      ./cmd/aegisctl
	go build -o $(DIST)/aegisd        ./cmd/aegisd
	go build -o $(DIST)/aegis-fetcher ./cmd/aegis-fetcher
	@echo "Built to $(DIST)/"

install: build
	mkdir -p $(HOME)/.local/bin
	cp $(DIST)/aegisctl      $(HOME)/.local/bin/aegisctl
	cp $(DIST)/aegisd        $(HOME)/.local/bin/aegisd
	cp $(DIST)/aegis-fetcher $(HOME)/.local/bin/aegis-fetcher
	@echo "Installed binaries to ~/.local/bin/"

	mkdir -p $(HOME)/.config/aegis
	@if [ ! -f "$(HOME)/.config/aegis/aegis.yaml" ]; then \
		cp config/aegis.yaml $(HOME)/.config/aegis/aegis.yaml; \
		echo "Created ~/.config/aegis/aegis.yaml"; \
	else \
		echo "Skipped ~/.config/aegis/aegis.yaml (already exists)"; \
	fi

uninstall:
	rm -f $(HOME)/.local/bin/aegisctl
	rm -f $(HOME)/.local/bin/aegisd
	rm -f $(HOME)/.local/bin/aegis-fetcher
	@echo "Removed binaries from ~/.local/bin/"
	@printf "Remove ~/.config/aegis/aegis.yaml? [y/N] " && read ans && [ "$$ans" = "y" ] && rm -rf $(HOME)/.config/aegis && echo "Removed ~/.config/aegis/" || echo "Config kept."

clean:
	rm -rf $(DIST)