# TODO — Mansio

## Phase 2: Code Review Panel (Next)

- [ ] Git 상태 감지: 워크스페이스 디렉토리의 `.git` 존재 여부 확인
- [ ] `git diff` 뷰어: 변경된 파일 목록 + 인라인 diff 표시
- [ ] `git log` 뷰어: 커밋 히스토리 브라우징
- [ ] `git status` 패널: staged/unstaged/untracked 파일 표시
- [ ] diff 구문 하이라이팅 (언어별 syntax highlight)

## UI/UX 개선

- [ ] 탭 드래그 정렬 (drag-to-reorder)
- [ ] 워크스페이스 드래그 정렬
- [ ] 터미널 검색 (Ctrl+Shift+F → 스크롤백 내 텍스트 검색)
- [ ] 터미널 분할 (horizontal/vertical split within a tab)
- [ ] 커스텀 테마 지원 (사용자 정의 색상)
- [ ] 폰트 크기 조절 (Ctrl+= / Ctrl+-)
- [ ] 워크스페이스별 아이콘/색상 라벨
- [ ] 터미널 탭 이름 자동 감지 (OSC title sequence)
- [ ] 키보드 단축키: Ctrl+T (새 탭), Ctrl+W (탭 닫기), Ctrl+1-9 (탭 전환)
- [ ] 전체 키보드 단축키 도움말 패널

## 인증 & 보안

- [ ] HTTPS/TLS 지원 (Let's Encrypt 자동 인증서 또는 자체 인증서)
- [ ] 멀티유저 지원 (사용자별 워크스페이스 격리)
- [ ] OAuth 연동 (GitHub, Google)
- [ ] 2FA (TOTP) 지원
- [ ] 세션 타임아웃 설정 (현재 7일 고정)

## 인프라 & 배포

- [ ] Docker Compose에 환경변수로 초기 비밀번호 설정 (`MANSIO_PASSWORD`)
- [ ] ARM64 + AMD64 멀티 아키텍처 Docker 이미지
- [ ] CI/CD 파이프라인 (GitHub Actions: 테스트 → 빌드 → Docker push)
- [ ] Helm chart (Kubernetes 배포)
- [ ] 자동 업데이트 알림

## 성능 & 안정성

- [ ] WebSocket 백프레셔: 대량 출력 시 프레임 드롭 방지
- [ ] xterm.js 인스턴스 풀 LRU 관리 (20개 초과 시 가장 오래된 탭 dispose)
- [ ] 서버 시작 시 orphan tmux 세션 정리 (DB에 없는 `lt_*` 세션 감지)
- [ ] 헬스체크 엔드포인트 개선 (tmux 서버 상태, DB 연결 상태 포함)
- [ ] 연결 끊김 시 UI 표시 (reconnecting 스피너)
- [ ] Go 테스트: tmux 매니저 통합 테스트
- [ ] Go 테스트: WebSocket 핸들러 테스트
- [ ] 프론트엔드 테스트: 컴포넌트 렌더링 테스트 (Sidebar, TabBar, LoginForm)
- [ ] E2E 테스트: Playwright 기반 전체 플로우 테스트

## 확장 기능

- [ ] 파일 업로드/다운로드 (터미널 내 drag & drop)
- [ ] 터미널 녹화 & 재생 (asciinema 호환)
- [ ] 알림: 장시간 작업 완료 시 브라우저 알림
- [ ] API 토큰 인증 (CLI/자동화 용도)
- [ ] 터미널 공유 (읽기 전용 URL 생성)
- [ ] 플러그인 시스템 (커스텀 패널 추가)
