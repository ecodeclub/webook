# 单元测试
.PHONY: ut
ut:
	@go test -race ./...

.PHONY: setup
setup:
	@sh ./script/setup.sh

.PHONY: lint
lint:
	golangci-lint run

.PHONY: fmt
fmt:
	@sh ./script/fmt.sh

.PHONY: tidy
tidy:
	@go mod tidy -v

.PHONY: check
check:
	@$(MAKE) --no-print-directory fmt
	@$(MAKE) --no-print-directory tidy

# e2e 测试
.PHONY: e2e
e2e:
	sh ./script/integrate_test.sh

.PHONY: e2e_up
e2e_up:
	docker compose -f script/integration_test_compose.yml up -d

.PHONY: e2e_down
e2e_down:
	docker compose -f script/integration_test_compose.yml down

.PHONY: mock
mock:
	@mockgen -source=./internal/web/token/generator/jwt.go -package=tokenmocks -destination=./internal/web/token/mocks/tokenGenerator.mock.go
	@mockgen -source=./internal/web/token/validator/token.go -package=tokenmocks -destination=./internal/web/token/mocks/tokenValidator.mock.go
	@mockgen -source=./internal/service/user.go -package=svcmocks -destination=./internal/service/mocks/user.mock.go
	@mockgen -source=./internal/service/email.go -package=svcmocks -destination=./internal/service/mocks/email.mock.go
	@mockgen -source=./internal/service/mail/types.go -package=mailmocks -destination=./internal/service/mail/mocks/mail.mock.go
	@mockgen -source=./internal/repository/user.go -package=repomocks -destination=./internal/repository/mocks/user.mock.go
	@mockgen -source=./internal/repository/dao/user.go -package=daomocks -destination=./internal/repository/dao/mocks/user.mock.go