FROM node:24-alpine AS build

WORKDIR /app

ARG VITE_GAME_SERVICE_URL=http://localhost:8080
ARG VITE_RATING_SERVICE_URL=http://localhost:8081
ENV VITE_GAME_SERVICE_URL=$VITE_GAME_SERVICE_URL
ENV VITE_RATING_SERVICE_URL=$VITE_RATING_SERVICE_URL

COPY frontend/package.json ./package.json
COPY frontend/vite.config.js ./vite.config.js
COPY frontend/index.html ./index.html
COPY frontend/src ./src

RUN npm install
RUN npm run build

FROM nginx:1.27-alpine

COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80
