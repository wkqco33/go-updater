BIN_NAME := go-updater
INSTALL_DIR := $(HOME)/.local/bin

.PHONY: build clean install uninstall test lint help

build:
	go build -o $(BIN_NAME) .

clean:
	rm -f $(BIN_NAME)

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BIN_NAME) $(INSTALL_DIR)/$(BIN_NAME)
	@echo "설치 완료: $(INSTALL_DIR)/$(BIN_NAME)"

uninstall:
	rm -f $(INSTALL_DIR)/$(BIN_NAME)
	@echo "삭제 완료: $(INSTALL_DIR)/$(BIN_NAME)"

test:
	go test ./...

lint:
	go vet ./...

help:
	@echo "사용 가능한 명령:"
	@echo "  make build     - 바이너리 빌드"
	@echo "  make clean     - 빌드 결과물 삭제"
	@echo "  make install   - 빌드 후 $(INSTALL_DIR)에 설치"
	@echo "  make uninstall - 설치된 바이너리 삭제"
	@echo "  make test      - 테스트 실행"
	@echo "  make lint      - go vet 실행"
