# Go Updater (gu)

`gu`는 시스템에 설치된 Go 언어를 여러 버전별로 관리하고, 최신 버전으로 빠르고 쉽게 업데이트할 수 있도록 도와주는 CLI 도구입니다.
`rustup`, `nvm`, `fnm` 등과 유사하게 여러 버전을 설치하고 필요에 따라 즉시 전환할 수 있는 기능을 제공합니다.

## 특징

- **버전 매니저 기능**: 여러 버전의 Go를 독립적으로 설치하고 관리합니다.
- **자동화된 최신 버전 감지**: `go.dev`의 공식 릴리스 정보를 바탕으로 안정적인 최신 버전을 자동으로 탐색합니다.
- **특정 버전 설치**: 최신 버전뿐만 아니라 사용자가 원하는 특정 버전(예: 1.20, 1.20.5)을 지정하여 설치할 수 있습니다.
- **손쉬운 버전 전환**: 심볼릭 링크를 통해 설치된 버전 간의 전환을 즉각적으로 수행합니다.
- **용량 관리**: 더 이상 사용하지 않는 구버전을 간편하게 삭제할 수 있습니다.
- **프라이빗 모듈 캐시**: 프라이빗 Go 모듈을 선캐시하고, 다른 프로젝트에서 동일 캐시를 재사용할 수 있습니다.
- **사용자 권한 최적화**: 기본적으로 사용자 디렉토리(`~/.go`)에 설치되어 관리자 권한(`sudo`)이 필요하지 않습니다.

## 설치 방법

소스 코드에서 직접 빌드하여 사용할 수 있습니다. 빌드에는 [Task](https://taskfile.dev)가 필요합니다.

```bash
git clone <repository_url>
cd go-updater
go mod tidy
task build
```

빌드된 바이너리를 `~/.local/bin`에 설치하려면:

```bash
task install
```

그 외 사용 가능한 task 명령어:

| 명령어 | 설명 |
|---|---|
| `task build` | 바이너리 빌드 |
| `task clean` | 빌드 결과물 삭제 |
| `task install` | 빌드 후 `~/.local/bin`에 설치 |
| `task uninstall` | 설치된 바이너리 삭제 |
| `task test` | 테스트 실행 |
| `task lint` | `go vet` 실행 |

## 사용 방법

### 글로벌 옵션

모든 명령에서 사용 가능한 옵션입니다.

```bash
# 디버그 로그 활성화
gu --debug <command>
```

| 플래그 | 설명 |
|---|---|
| `--debug` | 상세한 디버깅 로그를 stderr에 출력합니다 |

### 1. Go 설치 (`install`)

최신 버전 또는 특정 버전을 설치합니다. 설치 후 해당 버전이 즉시 활성화됩니다.

```bash
# 최신 안정 버전 설치
gu install

# 특정 마이너 버전의 최신 패치 버전 설치 (예: 1.20.x 중 최신)
gu install 1.20

# 특정 버전 설치
gu install 1.20.5

# 설치 디렉토리를 지정하여 설치 (기본값: ~/.go)
gu install -d /opt/go 1.21.0
```

| 플래그 | 단축 | 기본값 | 설명 |
|---|---|---|---|
| `--dir` | `-d` | `~/.go` | Go가 설치될 최상위 디렉토리 |

### 2. 설치된 목록 확인 (`list`)

로컬에 설치된 모든 Go 버전 목록과 현재 사용 중인 버전을 확인합니다.

```bash
gu list
```

### 3. 버전 전환 (`use`)

이미 설치된 다른 버전으로 즉시 전환합니다. 마이너 버전만 지정하면 해당 마이너 버전 중 가장 높은 패치 버전으로 전환합니다.

```bash
# 정확한 버전 지정
gu use 1.20.5

# 마이너 버전 지정 (1.20.x 중 가장 높은 패치로 전환)
gu use 1.20
```

### 4. 버전 삭제 및 정리 (`clean`)

설치된 버전을 삭제하여 디스크 용량을 확보합니다.

```bash
# 특정 버전 삭제
gu clean 1.20.5

# 현재 사용 중인 버전을 제외한 모든 버전 삭제
gu clean --unused

# 모든 Go 버전 및 관련 파일 삭제
gu clean --all

# go.dev에서 설치된 시스템 Go 삭제 (/usr/local/go 또는 C:\Go)
gu clean --system
```

| 플래그 | 설명 |
|---|---|
| `--all` | 모든 설치된 Go 버전을 삭제합니다 (복구 불가) |
| `--unused` | 현재 사용 중인 버전을 제외한 모든 버전을 삭제합니다 |
| `--system` | go.dev에서 설치된 시스템 Go(`/usr/local/go` 또는 `C:\Go`)를 삭제합니다 |

### 5. 버전 확인 (`version`)

`gu` 자체의 버전을 확인합니다.

```bash
gu version
```

## 프라이빗 모듈 캐시 사용

`gu private` 명령으로 프라이빗 모듈 캐시를 구성하고, 다른 프로젝트에서 동일 캐시를 재사용할 수 있습니다.

### Quick Start: 프라이빗 GitHub 라이브러리 사용하기

프라이빗 GitHub 저장소(예: `github.com/my-org/private-lib`)를 Go 라이브러리로 사용하는 전체 흐름입니다.

#### 사전 준비: Git 인증 설정

`gu`는 인증정보를 저장하지 않으므로, Git이 프라이빗 저장소에 접근할 수 있도록 먼저 설정해야 합니다.

```bash
# 방법 A: SSH 키 사용 (권장)
# ~/.gitconfig에 HTTPS → SSH 변환 설정
git config --global url."git@github.com:".insteadOf "https://github.com/"

# 방법 B: GitHub PAT(Personal Access Token) 사용
# ~/.netrc에 토큰 설정
echo "machine github.com login YOUR_USERNAME password ghp_YOUR_TOKEN" >> ~/.netrc
chmod 600 ~/.netrc
```

#### 전체 워크플로우

```bash
# 1. 설정 초기화
gu private config init
gu private config set --private "github.com/my-org/*"

# 2. 라이브러리 선캐시
gu private sync github.com/my-org/private-lib@v1.2.3

# 3. 소비 프로젝트에서 환경변수 적용 후 빌드
cd ~/projects/my-app
eval $(gu private env)
go mod tidy
go build ./...
```

> **팀/CI 공유 팁**: 동일한 캐시 디렉토리(`~/.go/private/modcache`)를 NFS, 볼륨 마운트 등으로 공유하면 팀원이나 CI에서 매번 프라이빗 저장소에 접근하지 않아도 됩니다.
> 네트워크 없이 빌드해야 하는 환경에서는 `eval $(gu private env --offline)`을 사용하세요.

---

### 상세 명령어 가이드

### 1) 설정 초기화 및 정책 설정

```bash
# 기본 설정 파일 생성 (~/.go/private/config.json)
gu private config init

# private 모듈 패턴 + 캐시 경로 설정
gu private config set \
  --private github.com/my-org/*,git.example.com/* \
  --cache-dir ~/.go/private/modcache

# GONOSUMDB, GONOPROXY 패턴을 별도로 지정
gu private config set \
  --nosumdb github.com/my-org/* \
  --noproxy github.com/my-org/*

# 현재 설정 확인
gu private config show
```

`private config set` 명령의 플래그:

| 플래그 | 설명 |
|---|---|
| `--private` | GOPRIVATE 패턴 (쉼표 구분) |
| `--cache-dir` | 모듈 캐시 디렉토리 경로 |
| `--nosumdb` | GONOSUMDB 패턴 (쉼표 구분) |
| `--noproxy` | GONOPROXY 패턴 (쉼표 구분) |

> `--nosumdb`, `--noproxy`를 지정하지 않으면 `--private`와 동일 패턴으로 자동 적용됩니다.

### 2) 모듈 선캐시(sync)

```bash
# 버전 고정 동기화
gu private sync github.com/my-org/private-lib@v1.2.3

# 버전 생략 시 latest 사용 (기본값)
gu private sync github.com/my-org/private-lib

# 여러 모듈을 한 번에 동기화
gu private sync github.com/my-org/repo1@v1.0.0 github.com/my-org/repo2@v2.1.0

# 재시도 횟수 지정
gu private sync --retries 5 github.com/my-org/private-lib@v1.2.3

# 동기화 소스 식별자 지정 (메타데이터에 기록됨)
gu private sync --source github-enterprise github.com/my-org/private-lib@v1.2.3
```

`private sync` 명령의 플래그:

| 플래그 | 기본값 | 설명 |
|---|---|---|
| `--retries` | `3` | 모듈 다운로드 재시도 횟수 |
| `--latest-if-missing` | `true` | 버전 미지정 시 latest 사용 |
| `--source` | `manual` | 동기화 소스 식별자 (예: `github-enterprise`) |

동기화 메타데이터는 `~/.go/private/metadata.json`에 기록됩니다.

### 3) 다른 프로젝트에서 캐시 재사용

```bash
# export 스크립트 출력
gu private env

# 오프라인 우선 모드(GOPROXY=off 포함)
gu private env --offline

# 현재 셸에 즉시 적용
eval $(gu private env)
```

출력된 환경변수를 소비 프로젝트 셸에 적용한 뒤 `go build`, `go test`를 실행하면 동일 캐시를 재사용할 수 있습니다.

| 플래그 | 기본값 | 설명 |
|---|---|---|
| `--offline` | `false` | `GOPROXY=off`를 함께 출력하여 오프라인 우선 모드로 설정 |

### 4) 캐시 정리

```bash
# N일 이전 파일 정리
gu private clean --stale-days 30

# 최대 용량(MB) 초과 시 오래된 파일부터 정리
gu private clean --max-size-mb 2048

# 캐시 전체 삭제
gu private clean --all
```

| 플래그 | 기본값 | 설명 |
|---|---|---|
| `--stale-days` | `0` | N일 이전의 오래된 파일만 삭제 |
| `--max-size-mb` | `0` | 캐시가 지정 MB를 초과할 경우 오래된 파일부터 삭제 |
| `--all` | `false` | 캐시 전체 및 메타데이터를 삭제 |

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

## 명령어 레퍼런스

| 명령어 | 설명 |
|---|---|
| `gu install [version]` | Go를 설치하거나 특정 버전으로 업데이트 |
| `gu list` | 설치된 Go 버전 목록 출력 |
| `gu use <version>` | 설치된 특정 버전의 Go로 전환 |
| `gu clean [version]` | 설치된 Go 버전 삭제 |
| `gu version` | `gu`의 버전 정보 출력 |
| `gu private config init` | 기본 프라이빗 캐시 설정 파일 생성 |
| `gu private config set` | 프라이빗 모듈 캐시 설정 값 갱신 |
| `gu private config show` | 현재 프라이빗 모듈 캐시 설정 출력 |
| `gu private sync <module[@version]> ...` | 프라이빗 모듈을 로컬 캐시에 동기화 |
| `gu private env` | 캐시 재사용용 환경변수 출력 |
| `gu private clean` | 프라이빗 모듈 캐시 정리 |
