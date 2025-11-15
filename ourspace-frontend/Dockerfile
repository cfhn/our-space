FROM node:lts AS build
WORKDIR /src
RUN corepack enable pnpm
COPY . .
RUN pnpm build-only

FROM nginx:mainline
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --link --from=build /src/dist /usr/share/nginx/html
