# Go Updater (gu)

`gu`는 시스템에 설치된 Go 언어를 여러 버전별로 관리하고, 최신 버전으로 빠르고 쉽게
업데이트할 수 있도록 도와주는 CLI 도구입니다. `rustup`, `nvm`, `fnm` 등과 유사하게
여러 버전을 설치하고 필요에 따라 즉시 전환할 수 있습니다.

문서: [CHANGELOG](CHANGELOG.md) · [CONTRIBUTING](CONTRIBUTING.md) · [SECURITY](SECURITY.md)

## 특징

- **버전 매니저 기능**: 여러 버전의 Go를 독립적으로 설치하고 관리합니다.
- **자동화된 최신 버전 감지**: `go.dev`의 공식 릴리스 정보를 바탕으로 안정적인 최신
  버전을 자동으로 탐색합니다.
- **특정 버전 설치**: 최신 버전뿐만 아니라 사용자가 원하는 특정 버전(예: `1.20`,
  `1.20.5`)을 지정하여 설치할 수 있습니다.
- **손쉬운 버전 전환**: 심볼릭 링크를 통해 설치된 버전 간의 전환을 즉각적으로 수행합니다.
- **용량 관리**: 더 이상 사용하지 않는 구버전을 간편하게 삭제할 수 있습니다.
- **프라이빗 모듈 캐시**: 프라이빗 Go 모듈을 선캐시하고, 다른 프로젝트에서 동일 캐시를
  재사용할 수 있습니다.
- **사용자 권한 최적화**: 기본적으로 사용자 디렉토리(`~/.go`)에 설치되어 관리자
  권한(`sudo`)이 필요하지 않습니다.
- **스크립트 친화적**: 결과는 stdout, 진행·오류는 stderr로 분리하고, `--json`,
  `--yes`, `--no-input`, `--dry-run`, 고유한 종료 코드를 제공합니다.

## 설치 방법

미리 빌드된 바이너리는 GitHub Releases에서 받을 수 있습니다. `ppm`을 사용하면 현재
OS와 CPU 아키텍처에 맞는 릴리스를 자동으로 설치할 수 있습니다.

```bash
ppm install wkqco33/go-updater
```

소스 코드에서 직접 빌드하려면 Go 1.26.8 이상과 [Task](https://taskfile.dev)가
필요합니다.

```bash
git clone https://github.com/wkqco33/go-updater.git
cd go-updater
task build
```

빌드된 바이너리를 `~/.local/bin`에 설치하려면:

```bash
task install
```

지원 플랫폼은 Linux (amd64/arm64), macOS (amd64/arm64), Windows (amd64/arm64)입니다.
직접 다운로드하는 경우 GitHub Releases의 플랫폼별 아카이브와 `.sha256` 파일을 함께
확인하세요.

다운로드한 Linux 아카이브의 무결성은 다음처럼 확인할 수 있습니다.

```bash
sha256sum -c gu_linux_amd64.tar.gz.sha256
```

macOS에서는 `shasum -a 256`을, Windows에서는 PowerShell의 `Get-FileHash`를 사용해
SHA-256 값을 비교하세요. 릴리스는 CI에서만 빌드되며 빌드 provenance 증명이 함께
게시됩니다.

```bash
gh attestation verify gu_linux_amd64.tar.gz --repo wkqco33/go-updater
```

그 외 사용 가능한 task 명령어:

| 명령어 | 설명 |
| --- | --- |
| `task build` | 바이너리 빌드 |
| `task clean` | 빌드 결과물 삭제 |
| `task install` | 빌드 후 `~/.local/bin`에 설치 |
| `task uninstall` | 설치된 바이너리 삭제 |
| `task test` | 테스트 실행 |
| `task test-race` | race detector와 함께 테스트 실행 |
| `task coverage` | 커버리지 리포트 생성 |
| `task verify` | gofmt, vet, 테스트 전체 실행 |
| `task lint` | `go vet` 실행 |

## 사용 방법

### 글로벌 옵션

모든 명령에서 사용 가능한 옵션입니다.

```bash
gu --debug install 1.21.0
gu clean --all --yes
gu list --json
```

| 플래그 | 단축 | 설명 |
| --- | --- | --- |
| `--debug` | | 상세한 디버깅 로그를 stderr에 출력합니다 |
| `--yes` | `-y` | 확인 프롬프트를 자동 승인합니다 (비대화형/CI) |
| `--no-input` | | 모든 대화형 프롬프트를 금지합니다 |
| `--quiet` | `-q` | 진행·상태 메시지를 출력하지 않습니다 |
| `--no-color` | | 색상 출력을 끕니다 (`NO_COLOR` 환경변수와 동일) |
| `--dry-run` | `-n` | 삭제·설치를 수행하지 않고 계획만 출력합니다 |
| `--home` | | gu가 버전과 캐시를 관리할 루트 (기본값: `$GU_HOME` 또는 `~/.go`) |
| `--version` | | `gu` 버전을 출력합니다 (`gu version`과 동일) |

> `install`의 `-d`/`--dir`는 설치 루트를 지정하는 단축 플래그입니다. `--debug`에는
> 단축 플래그가 없습니다.

### 출력과 종료 코드

CLI는 스크립트에서 안전하게 쓸 수 있도록 스트림과 종료 코드를 분리합니다.

- **stdout**: 결과 (버전 목록, JSON, `export` 문, 설치 완료 요약)
- **stderr**: 진행·상태·경고·오류 메시지와 확인 프롬프트
- 다운로드 진행률은 stderr가 TTY이고 `--quiet`가 아닐 때만 표시합니다.

| 종료 코드 | 의미 |
| --- | --- |
| `0` | 성공 (사용자가 확인 프롬프트에서 취소한 경우 포함) |
| `1` | 실행 오류 (네트워크, 파일 시스템, 체크섬, 외부 명령 실패) |
| `2` | 사용법 오류 (알 수 없는 명령/플래그, 인자 개수, 확인 불가) |

### 환경 변수

| 변수 | 설명 |
| --- | --- |
| `GU_HOME` | 버전·`current`·프라이빗 상태의 루트 디렉토리 (기본값: `~/.go`) |
| `XDG_CACHE_HOME` | 설정 시 프라이빗 모듈 캐시 기본 위치가 `$XDG_CACHE_HOME/go-updater/modcache`가 됩니다 |
| `NO_COLOR` | 설정 시 색상 출력을 끕니다 |
| `SHELL` | 설치 후 PATH 설정 안내에 사용할 셸 설정 파일을 결정합니다 |

`--home`이 지정되면 `GU_HOME`보다 우선하며, `install -d`는 `install` 명령에 한해
같은 값을 지정합니다.

### 1. Go 설치 (`install`)

최신 버전 또는 특정 버전을 설치합니다. 설치 후 해당 버전이 즉시 활성화됩니다.

```bash
# 최신 안정 버전 설치
gu install

# 특정 마이너 버전의 최신 안정 패치 버전 설치 (예: 1.20.x 중 최신)
gu install 1.20

# 특정 버전 설치
gu install 1.20.5

# 설치 디렉토리를 지정하여 설치 (기본값: ~/.go)
gu install -d /opt/go 1.21.0

# 네트워크 요청 전에 어떤 버전/URL을 설치할지 확인
gu install --dry-run 1.21.0
```

- 마이너 버전만 지정하면 해당 마이너의 **안정 릴리스** 중 가장 높은 패치를 선택합니다.
  `1.2`는 `go1.2.x`만 선택하며 `go1.20.x`나 `go1.27.x`를 선택하지 않습니다.
- `--dry-run`은 버전 확인까지만 수행하고 다운로드·설치는 하지 않습니다.

| 플래그 | 단축 | 기본값 | 설명 |
| --- | --- | --- | --- |
| `--dir` | `-d` | `$GU_HOME` 또는 `~/.go` | Go가 설치될 최상위 디렉토리 |

### 2. 설치된 목록 확인 (`list`)

로컬에 설치된 모든 Go 버전 목록과 현재 사용 중인 버전을 확인합니다.

```bash
gu list
gu list --json
```

`--json` 출력은 `{"root": "...", "current": "...", "versions": [{"name": "...",
"active": true}]}` 형식입니다. TTY에서 활성 버전은 색상으로 강조되며, `--no-color`나
`NO_COLOR`로 끌 수 있습니다.

### 3. 버전 전환 (`use`)

이미 설치된 다른 버전으로 즉시 전환합니다. 마이너 버전만 지정하면 해당 마이너 중
가장 높은 설치 패치 버전으로 전환합니다.

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

# 모든 Go 버전 및 관련 파일 삭제 (확인 프롬프트)
gu clean --all

# 비대화형 환경에서 확인 없이 실행
gu clean --all --yes

# 삭제 대상만 확인
gu clean --all --dry-run

# go.dev에서 설치된 시스템 Go 삭제
gu clean --system
```

| 플래그 | 설명 |
| --- | --- |
| `--all` | 모든 설치된 Go 버전을 삭제합니다 (복구 불가) |
| `--unused` | 현재 사용 중인 버전을 제외한 모든 버전을 삭제합니다 |
| `--system` | go.dev에서 설치된 시스템 Go를 삭제합니다 |

#### 확인 정책

- 여러 버전을 한 번에 지우거나 시스템 전체를 건드리는 작업(`--all`, `--unused`,
  `--system`)은 실행 전에 확인을 받습니다.
- 확인 프롬프트는 stdin이 TTY일 때만 표시됩니다. 파이프·CI처럼 대화형 입력이 없는
  환경에서는 `--yes`로 승인하거나 `--dry-run`으로 계획만 확인하세요. 이때 프롬프트를
  띄울 수 없으면 종료 코드 `2`로 실패합니다.
- 단일 버전 삭제(`gu clean 1.20.5`)는 대상을 명시하므로 확인을 받지 않습니다.
- 설치되어 있지 않거나 현재 활성 버전인 경우에는 삭제하지 않고 stderr에 경고만 남기고
  종료 코드 `0`으로 끝납니다(반복 실행해도 안전).

**`--system` 플래그 동작:**

- **macOS**: go.dev `.pkg` 인스톨러가 남기는 세 가지 흔적(`/usr/local/go`,
  `/etc/paths.d/go`, `pkgutil` 설치 리시트)을 각각 감지하여 실제로 남아있는 것만
  정리합니다. 세 흔적 중 일부만 남아있어도(예: 디렉토리는 이미 지웠지만 리시트가 남은
  경우) 정상적으로 감지·정리합니다. root 권한이 필요한 항목은 실행할 명령어를 먼저
  보여준 뒤 확인을 거쳐 `sudo`로 실행하며, 이때 sudo 비밀번호를 물을 수 있습니다.
  `/etc/paths.d/go`를 삭제한 경우 **새 터미널 세션부터** PATH 변경이 반영됩니다.
  Homebrew로 설치된 Go는 감지만 하고 삭제하지 않으며, `brew uninstall go`를
  안내합니다.
- **Windows**: `C:\Go` 경로와 winget 설치를 확인해 삭제합니다.

### 5. 버전 확인 (`version`)

`gu` 자체의 버전을 확인합니다.

```bash
gu version
gu version --json
gu --version
```

릴리스 바이너리에서는 명령어 버전이 해당 Git tag와 일치합니다. 소스에서 직접 빌드한
경우 버전은 `dev`로 표시됩니다.

## 프라이빗 모듈 캐시 사용

`gu private` 명령으로 프라이빗 모듈 캐시를 구성하고, 다른 프로젝트에서 동일 캐시를
재사용할 수 있습니다.

### Quick Start: 프라이빗 GitHub 라이브러리 사용하기

프라이빗 GitHub 저장소(예: `github.com/my-org/private-lib`)를 Go 라이브러리로 사용하는
전체 흐름입니다.

#### 사전 준비: Git 인증 설정

`gu`는 인증정보를 저장하지 않으므로, Git이 프라이빗 저장소에 접근할 수 있도록 먼저
설정해야 합니다.

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

> **팀/CI 공유 팁**: 동일한 캐시 디렉토리(`~/.go/private/modcache`)를 NFS, 볼륨
> 마운트 등으로 공유하면 팀원이나 CI에서 매번 프라이빗 저장소에 접근하지 않아도
> 됩니다. 네트워크 없이 빌드해야 하는 환경에서는
> `eval $(gu private env --offline)`을 사용하세요.

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
| --- | --- |
| `--private` | GOPRIVATE 패턴 (쉼표 구분) |
| `--cache-dir` | 모듈 캐시 디렉토리 경로 |
| `--nosumdb` | GONOSUMDB 패턴 (쉼표 구분) |
| `--noproxy` | GONOPROXY 패턴 (쉼표 구분) |

> `--nosumdb`, `--noproxy`를 지정하지 않으면 `--private`와 동일 패턴으로 자동 적용됩니다.

설정 파일 위치는 `$GU_HOME/private/config.json`이며, `GU_HOME`을 지정하지 않으면
`~/.go/private/config.json`입니다.

### 2) 모듈 선캐시(sync)

```bash
# 버전 고정 동기화
gu private sync github.com/my-org/private-lib@v1.2.3

# 버전 생략 시 latest 사용 (기본값)
gu private sync github.com/my-org/private-lib

# 여러 모듈을 한 번에 동기화
gu private sync github.com/my-org/repo1@v1.0.0 github.com/my-org/repo2@v2.1.0

# 재시도 횟수 지정 (실패 시 지수 백오프 + 지터로 재시도)
gu private sync --retries 5 github.com/my-org/private-lib@v1.2.3

# 동기화 소스 식별자 지정 (메타데이터에 기록됨)
gu private sync --source github-enterprise github.com/my-org/private-lib@v1.2.3
```

`private sync` 명령의 플래그:

| 플래그 | 기본값 | 설명 |
| --- | --- | --- |
| `--retries` | `3` | 모듈 다운로드 재시도 횟수 |
| `--latest-if-missing` | `true` | 버전 미지정 시 latest 사용 |
| `--source` | `manual` | 동기화 소스 식별자 (예: `github-enterprise`) |

동기화 메타데이터는 `$GU_HOME/private/metadata.json`(기본값
`~/.go/private/metadata.json`)에 기록됩니다.

### 3) 다른 프로젝트에서 캐시 재사용

```bash
# export 스크립트 출력
gu private env

# 오프라인 우선 모드(GOPROXY=off 포함)
gu private env --offline

# 현재 셸에 즉시 적용
eval $(gu private env)
```

출력은 POSIX 단일 인용부호로 이스케이프되므로 경로에 `$`, 백틱, 작은따옴표가 있어도
`eval` 시 확장되지 않습니다. 출력된 환경변수를 소비 프로젝트 셸에 적용한 뒤
`go build`, `go test`를 실행하면 동일 캐시를 재사용할 수 있습니다.

| 플래그 | 기본값 | 설명 |
| --- | --- | --- |
| `--offline` | `false` | `GOPROXY=off`를 함께 출력하여 오프라인 우선 모드로 설정 |

### 4) 캐시 정리

```bash
# N일 이전 파일 정리
gu private clean --stale-days 30

# 최대 용량(MB) 초과 시 오래된 파일부터 정리
gu private clean --max-size-mb 2048

# 캐시 전체 삭제 (확인 프롬프트)
gu private clean --all

# 비대화형 실행 / 계획만 확인
gu private clean --all --yes
gu private clean --all --dry-run
```

| 플래그 | 기본값 | 설명 |
| --- | --- | --- |
| `--stale-days` | `0` | N일 이전의 오래된 파일만 삭제 |
| `--max-size-mb` | `0` | 캐시가 지정 MB를 초과할 경우 오래된 파일부터 삭제 |
| `--all` | `false` | 캐시 전체 및 메타데이터를 삭제 |

`--dry-run`은 삭제 대상 개수와 용량만 계산해 보여주고 실제 파일은 건드리지 않습니다.

### 보안 주의사항

- 이 기능은 인증정보(토큰/패스워드)를 설정 파일에 저장하지 않습니다.
- 인증은 SSH 에이전트, 환경변수, `git credential helper`, OS 키체인 등 외부 비밀
  저장소를 사용하세요.
- CI 로그에 민감정보가 출력되지 않도록 `go env`, `git remote -v` 출력을 그대로 노출하지
  마세요.

## 환경 변수 설정 (PATH)

버전 매니저 기능을 정상적으로 사용하려면, 환경 변수 `PATH`에 아래 경로를 추가해야
합니다. 설치 완료 후 출력되는 가이드에 따라 `.bashrc` 또는 `.zshrc`에 추가하세요.

```bash
# ~/.go/current/bin 경로를 PATH에 추가 (이 경로는 심볼릭 링크이므로 버전 전환 시에도 고정됩니다)
export PATH=$PATH:~/.go/current/bin
```

`--home`/`GU_HOME`으로 루트를 바꿨다면 해당 루트의 `current/bin`을 추가하세요.

## 명령어 레퍼런스

| 명령어 | 설명 |
| --- | --- |
| `gu install [version]` | Go를 설치하거나 특정 버전으로 업데이트 |
| `gu list [--json]` | 설치된 Go 버전 목록 출력 |
| `gu use <version>` | 설치된 특정 버전의 Go로 전환 |
| `gu clean [version]` | 설치된 Go 버전 삭제 (`--all`, `--unused`, `--system`) |
| `gu version [--json]` | `gu`의 버전 정보 출력 |
| `gu private config init` | 기본 프라이빗 캐시 설정 파일 생성 |
| `gu private config set` | 프라이빗 모듈 캐시 설정 값 갱신 |
| `gu private config show` | 현재 프라이빗 모듈 캐시 설정 출력 |
| `gu private sync <module[@version]> ...` | 프라이빗 모듈을 로컬 캐시에 동기화 |
| `gu private env` | 캐시 재사용용 환경변수 출력 |
| `gu private clean` | 프라이빗 모듈 캐시 정리 |
