# syntax=docker/dockerfile:1
FROM node:24-slim

RUN \
  --mount=type=cache,target=/var/lib/apt,sharing=locked \
  --mount=type=cache,target=/var/cache/apt,sharing=locked \
  apt-get update -qq && apt-get install -y build-essential default-libmysqlclient-dev openssl \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /home/webapp

COPY package.json package-lock.json tsconfig.json ./
RUN npm ci

COPY . .
RUN npm run build

CMD [ "node", "dist/app.js" ]
