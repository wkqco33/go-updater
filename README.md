# Go Updater (go-updater)

`go-updater`는 시스템에 설치된 Go 언어를 여러 버전별로 관리하고, 최신 버전으로 빠르고 쉽게 업데이트할 수 있도록 도와주는 CLI 도구입니다.
`rustup`, `nvm`, `fnm` 등과 유사하게 여러 버전을 설치하고 필요에 따라 즉시 전환할 수 있는 기능을 제공합니다.

## 특징

- **버전 매니저 기능**: 여러 버전의 Go를 독립적으로 설치하고 관리합니다.
- **자동화된 최신 버전 감지**: `go.dev`의 공식 릴리스 정보를 바탕으로 안정적인 최신 버전을 자동으로 탐색합니다.
- **특정 버전 설치**: 최신 버전뿐만 아니라 사용자가 원하는 특정 버전(예: 1.20, 1.20.5)을 지정하여 설치할 수 있습니다.
- **손쉬운 버전 전환**: 심볼릭 링크를 통해 설치된 버전 간의 전환을 즉각적으로 수행합니다.
- **용량 관리**: 더 이상 사용하지 않는 구버전을 간편하게 삭제할 수 있습니다.
- **사용자 권한 최적화**: 기본적으로 사용자 디렉토리(`~/.go`)에 설치되어 관리자 권한(`sudo`)이 필요하지 않습니다.

## 설치 방법

소스 코드에서 직접 빌드하여 사용할 수 있습니다.

```bash
git clone <repository_url>
cd go-updater
go mod tidy
go build -o go-updater
```

## 사용 방법

### 1. Go 설치 (`install`)

최신 버전 또는 특정 버전을 설치합니다. 설치 후 해당 버전이 즉시 활성화됩니다.

```bash
# 최신 안정 버전 설치
./go-updater install

# 특정 마이너 버전의 최신 패치 버전 설치 (예: 1.20.x 중 최신)
./go-updater install 1.20

# 특정 버전 설치
./go-updater install 1.20.5
```

### 2. 설치된 목록 확인 (`list`)

로컬에 설치된 모든 Go 버전 목록과 현재 사용 중인 버전을 확인합니다.

```bash
./go-updater list
```

### 3. 버전 전환 (`use`)

이미 설치된 다른 버전으로 즉시 전환합니다.

```bash
./go-updater use 1.20
```

### 4. 버전 삭제 및 정리 (`clean`)

설치된 버전을 삭제하여 디스크 용량을 확보합니다.

```bash
# 특정 버전 삭제
./go-updater clean 1.20.5

# 현재 사용 중인 버전을 제외한 모든 버전 삭제
./go-updater clean --unused

# 모든 Go 버전 및 관련 파일 삭제
./go-updater clean --all
```

### 5. 버전 확인 (`version`)

`go-updater` 자체의 버전을 확인합니다.

```bash
./go-updater version
```

## 프라이빗 모듈 캐시 사용

`go-updater private` 명령으로 프라이빗 모듈 캐시를 구성하고, 다른 프로젝트에서 동일 캐시를 재사용할 수 있습니다.

### 1) 설정 초기화 및 정책 설정

```bash
# 기본 설정 파일 생성 (~/.go/private/config.json)
./go-updater private config init

# private 모듈 패턴 + 캐시 경로 설정
./go-updater private config set \
  --private github.com/my-org/*,git.example.com/* \
  --cache-dir ~/.go/private/modcache

# 현재 설정 확인
./go-updater private config show
```

> `GONOSUMDB`, `GONOPROXY`를 지정하지 않으면 `GOPRIVATE`와 동일 패턴으로 자동 적용됩니다.

### 2) 모듈 선캐시(sync)

```bash
# 버전 고정 동기화
./go-updater private sync github.com/my-org/private-lib@v1.2.3

# 버전 생략 시 latest 사용 (기본값)
./go-updater private sync github.com/my-org/private-lib

# 재시도 횟수 지정
./go-updater private sync --retries 5 github.com/my-org/private-lib@v1.2.3
```

동기화 메타데이터는 `~/.go/private/metadata.json`에 기록됩니다.

### 3) 다른 프로젝트에서 캐시 재사용

```bash
# export 스크립트 출력
./go-updater private env

# 오프라인 우선 모드(GOPROXY=off 포함)
./go-updater private env --offline
```

출력된 환경변수를 소비 프로젝트 셸에 적용한 뒤 `go build`, `go test`를 실행하면 동일 캐시를 재사용할 수 있습니다.

### 4) 캐시 정리

```bash
# N일 이전 파일 정리
./go-updater private clean --stale-days 30

# 최대 용량(MB) 초과 시 오래된 파일부터 정리
./go-updater private clean --max-size-mb 2048

# 캐시 전체 삭제
./go-updater private clean --all
```

### 보안 주의사항

- 이 기능은 인증정보(토큰/패스워드)를 설정 파일에 저장하지 않습니다.
- 인증은 SSH 에이전트, 환경변수, `git credential helper`, OS 키체인 등 외부 비밀 저장소를 사용하세요.
- CI 로그에 민감정보가 출력되지 않도록 `go env`, `git remote -v` 출력을 그대로 노출하지 마세요.

## 환경 변수 설정 (PATH)

버전 매니저 기능을 정상적으로 사용하려면, 환경 변수 `PATH`에 아래 경로를 추가해야 합니다.
설치 완료 후 출력되는 가이드에 따라 `.bashrc` 또는 `.zshrc`에 추가하세요.

```bash
# ~/.go/current/bin 경로를 PATH에 추가 (이 경로는 심볼릭 링크이므로 버전 전환 시에도 고정됩니다)
export PATH=$PATH:~/.go/current/bin
```
