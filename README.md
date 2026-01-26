# weeky

> Weekly report, without the hassle

주간업무보고 PPT를 자동으로 생성해주는 웹 도구입니다.

## Quick Start

### Development

```bash
# Backend
cd backend
go run ./cmd/server

# Frontend (새 터미널)
cd frontend
npm install
npm run dev
```

- Frontend: http://localhost:3000
- Backend: http://localhost:8080

### Docker

```bash
docker-compose up -d
```

http://localhost:8080 접속

## Features

- 웹 UI에서 간단히 텍스트 입력
- 금주실적, 차주계획, 이슈 항목별 관리
- 템플릿 기반 PPT 자동 생성
- 파일명 자동 생성 (팀명_이름_주간보고_날짜.pptx)

## Tech Stack

- Backend: Go + Fiber
- Frontend: React + Tailwind CSS
- PPT: PptxGenJS
- Database: SQLite

## License

MIT
