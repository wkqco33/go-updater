# 변경 이력 (Changelog)

이 문서는 [Keep a Changelog](https://keepachangelog.com/ko/1.1.0/) 형식과
[유의적 버전](https://semver.org/lang/ko/)을 따릅니다. 여기의 버전 번호는 Git 태그이자
릴리스 바이너리의 버전(`gu --version`)과 일치합니다.

## [Unreleased]

## [0.2.0] - 2026-09-12

### Added

- 전역 플래그를 추가했습니다: `--yes`/`-y`, `--no-input`, `--quiet`/`-q`,
  `--no-color`, `--dry-run`/`-n`, `--home`.
- `gu list --json`, `gu version --json` 기계 판독 출력을 추가했습니다.
- 설정·데이터 위치를 `GU_HOME`, 프라이빗 모듈 캐시 기본 위치를 `XDG_CACHE_HOME`으로
  재정의할 수 있습니다.
- `gu install`, `gu clean`, `gu private clean`에 `--dry-run` 계획 출력을 추가했습니다.
- 서브커맨드 도움말에 전체 실행 경로와 예시·문서 링크를 추가했습니다.
- CI에 태그 push 테스트, fuzz 스모크, `govulncheck`, Dependabot을 추가했습니다.
- 릴리스에 빌드 provenance 증명(SLSA 계열 attestation)을 추가했습니다. 아카이브
  내부 레이아웃은 기존 설치 도구 호환을 위해 바이너리 단일 파일로 유지합니다.

### Changed

- 진행·상태 메시지는 stderr, 결과는 stdout으로 분리했습니다. 다운로드 진행률은 stderr가
  TTY이고 `--quiet`가 아닐 때만 출력합니다.
- 여러 버전을 지우는 파괴적 작업(`clean --all`, `clean --unused`, `clean --system`,
  `private clean --all`)은 확인 프롬프트를 거치며, 비대화형 환경에서는 `--yes`를
  요구합니다.
- 종료 코드를 분리했습니다: 성공 `0`, 실행 오류 `1`, 사용법 오류 `2`.
- `gu install`, `gu list`, `gu use`, `gu clean`이 `--home`/`GU_HOME` 루트를 공유합니다.
- 릴리스 파이프라인이 빌드 전에 테스트를 실행하고, 체크아웃 자격 증명을 남기지 않으며,
  공유 빌드 캐시를 사용하지 않습니다.
- 문서 파일 이름을 `CHANGE_LOG.md`에서 `CHANGELOG.md`로 변경했습니다.

### Fixed

- `gu install 1.2`가 최신 버전(예: `go1.27.x`)을 설치하던 버전 접두사 매칭 버그를
  수정했습니다. 부분 버전 요청은 같은 마이너의 안정 패치만 선택합니다.
- `gu clean <설치되지 않은 버전>`과 활성 버전 삭제 시도가 결과 대신 stderr 경고로
  출력되도록 수정했습니다.
- `gu private env` 출력을 POSIX 단일 인용부호로 이스케이프해 `eval` 시 `$`·백틱 확장을
  차단했습니다.
- `--dir`로 설치한 루트를 `list`/`use`/`clean`이 보지 못하던 불일치를 `--home`으로
  정리했습니다.

### Security

- 아카이브 추출은 경로 탈출과 비정규 엔트리를 거부합니다(기존 동작 유지, 테스트 보강).
- 외부 명령 실행 대상을 `go`로 고정했습니다.
- 캐시 전체 삭제가 파일 트리를 두 번 순회하지 않고 한 번에 측정·삭제하도록 변경했습니다.

## [0.1.5] - 2026-08-23

### Added

- 플랫폼별 릴리스 아카이브(Linux/macOS/Windows, amd64/arm64)와 SHA-256 checksum을
  추가했습니다.
- `internal/cli` 어댑터로 출력·홈·HTTP 경계를 주입할 수 있게 해 테스트를 강화했습니다.

### Changed

- TDD 커버리지를 확대하고 플랫폼 설치·아카이브 추출·fuzz 테스트를 추가했습니다.
- Go 파일 줄바꿈을 LF로 고정해 Windows에서도 `gofmt` 검사가 일관되게 통과합니다.

### Fixed

- 릴리스 생성이 저장소 체크아웃에 의존하지 않도록 수정했습니다.

## [0.1.4] - 2026-08-02

### Changed

- `install` 명령을 리팩터링하고 테스트를 추가했습니다.

## [0.1.3] - 2026-08-02

### Added

- winget으로 설치된 Go 감지·제거와 Windows 정션 폴백을 추가했습니다.

### Fixed

- Windows에서 기존 `current` 링크 교체가 실패할 때 실행 명령을 함께 출력하도록
  수정했습니다.

## [0.1.2] - 2026-08-01

### Added

- `private` 명령 그룹(`config init|set|show`, `sync`, `env`, `clean`)과 프라이빗 모듈
  캐시 기능을 추가했습니다.
- `clean --system`이 macOS에서 go.dev `.pkg` 설치 흔적(`/usr/local/go`,
  `/etc/paths.d/go`, `pkgutil` 리시트)을 개별 감지·정리합니다.
- Homebrew로 설치된 Go는 감지만 하고 `brew uninstall go`를 안내합니다.

### Changed

- 빌드·테스트 관리 도구를 `Makefile`에서 `Taskfile.yml`로 전환하고 바이너리 이름을
  `gu`로 변경했습니다.
- README와 명령어 레퍼런스를 갱신했습니다.

### Removed

- 기존 설치 기능을 `install` 명령으로 통합했습니다.

## [0.1.1] - 2026-03-18

### Added

- Go 버전 파싱·비교 유틸리티와 다운로드 진행률 표시를 추가했습니다.

## [0.1.0] - 2026-03-18

### Added

- `go.dev/dl/?mode=json` 기반 릴리스 조회, OS/아키텍처 매칭, 다운로드·SHA256 검증·
  압축 해제를 구현했습니다.
- `version` 명령과 `--debug` 로깅 플래그를 추가했습니다.
