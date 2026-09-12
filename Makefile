.PHONY: templ-generate templ-check css-check web-mock web-test

TEMPL_VERSION := v0.3.1020

templ-generate:
	go run github.com/a-h/templ/cmd/templ@$(TEMPL_VERSION) generate ./web/...

templ-check:
	go run github.com/a-h/templ/cmd/templ@$(TEMPL_VERSION) generate --check ./web/...

css-check:
	@test -s web/assets/css/app.css
	@test -s web/assets/css/theme.css

web-mock:
	WEB_UI_MODE=mock go run ./cmd/web

web-test:
	go test ./internal/web/... ./web/...
