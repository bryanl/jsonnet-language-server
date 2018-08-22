help:  ## Show help messages for make targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[32m%-30s\033[0m %s\n", $$1, $$2}'

jlsclient: ## build jlsclient
	go build -o jlsclient ./cmd/jlsclient/main.go


.PHONY: jlsclient help