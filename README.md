# weeky

> Weekly report, without the hassle

주간업무보고 PPT를 자동으로 생성해주는 멀티유저 웹 도구입니다.

## Features

- 외부 서비스 연동 (GitHub, GitLab, Jira, Hiworks)
- AI 기반 보고서 자동 생성
- PPT 미리보기 및 편집 후 다운로드
- 멀티유저 지원 (초대 코드 기반 가입, 사용자별 데이터 격리)

## 사용법

### 1. 회원가입 / 로그인
- 첫 번째 가입자는 자동으로 관리자가 됩니다
- 이후 팀원은 관리자가 발급한 초대 코드로 가입합니다

### 2. 설정
- **설정** 탭에서 사용할 서비스의 토큰을 등록합니다
- GitHub Personal Access Token, Jira API Token 등

### 3. 보고서 작성
- **보고서 작성** 탭에서 팀명, 작성자, 기간을 입력합니다
- **데이터 연동** 버튼을 눌러 GitHub 커밋, Jira 이슈 등을 자동으로 수집합니다
- **AI 자동생성** 버튼을 누르면 수집된 데이터를 기반으로 금주실적/차주계획이 자동 작성됩니다

### 4. 편집 및 다운로드
- AI가 생성한 내용을 확인하고, 필요하면 직접 수정합니다
- **PPT 미리보기**로 슬라이드를 확인한 뒤 **다운로드** 버튼을 누르면 완성된 PPT 파일을 받을 수 있습니다

## Quick Start

### Docker (권장)

```bash
docker pull jiin724/weeky:latest
docker compose up -d
```

http://localhost:8080 접속

### 환경변수

| 변수 | 설명 | 필수 |
|------|------|------|
| `ENCRYPTION_KEY` | AES-256 암호화 키 (32자 이상) | O |
| `JWT_SECRET` | JWT 서명 키 | O |
| `DB_PATH` | SQLite DB 경로 (기본: `./weeky.db`) | |
| `PORT` | 서버 포트 (기본: `8080`) | |

### Development

```bash
# Backend
cd backend
cp .env.example .env  # ENCRYPTION_KEY 설정
go run ./cmd/server

# Frontend
cd frontend
npm install
npm run dev
```

## Tech Stack

- Backend: Go + Fiber + SQLite
- Frontend: React + TypeScript + Tailwind CSS
- Deploy: Docker (multi-stage build)

## License

MIT
