#!/bin/sh
set -e

VERSION=$(git describe --tags --always)
REVISION=$(git rev-parse HEAD)
DATE=$(TZ=Asia/Seoul date +"%Y-%m-%d %H:%M %Z")
GO=$(go version | awk '{print $3}')

# 현재 폴더명 추출
APP=$(basename "$PWD")

# 소문자 변수명으로 변경 (구조체는 대문자, 주입 변수는 소문자)
go build -ldflags "-X 'main.Version=$VERSION' \
    -X 'main.Revision=$REVISION' \
    -X 'main.Date=$DATE' \
    -X 'main.Go=$GO'" -o "$APP"

./"$APP"