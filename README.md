# weeky

> Weekly report, without the hassle

주간업무보고 PPT를 자동으로 생성해주는 멀티유저 웹 도구입니다.

## Features

- 웹 UI에서 금주실적, 차주계획, 이슈 입력 → PPT 자동 생성
- 외부 서비스 연동 (GitHub, GitLab, Jira, Hiworks)
- AI 기반 보고서 자동 생성
- 멀티유저 지원 (초대 코드 기반 가입, 사용자별 데이터 격리)
- JWT 인증 + Refresh Token 자동 갱신

## Quick Start

### Docker (권장)

```bash
docker pull jiin724/weeky:latest
docker compose up -d
```

http://localhost:8080 접속 → 첫 번째 가입자가 관리자

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
