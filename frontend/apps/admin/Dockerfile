# Build context precisa ser `frontend/` (não `frontend/apps/admin/`), para enxergar o
# workspace pnpm inteiro (packages/shared é uma dependência real do app). No EasyPanel:
# Build Context = frontend, Dockerfile Path = apps/admin/Dockerfile.
FROM node:22-alpine AS build
RUN corepack enable && corepack prepare pnpm@9 --activate
WORKDIR /repo
COPY pnpm-workspace.yaml pnpm-lock.yaml ./
COPY packages ./packages
COPY apps ./apps
RUN pnpm install --frozen-lockfile
RUN pnpm --filter @p5wellness/admin build

FROM nginx:1.27-alpine
COPY apps/admin/nginx.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /repo/apps/admin/dist /usr/share/nginx/html
# Domínio público da API (sem esquema) — só precisa mudar se o domínio real for
# diferente deste (ver deploy/EASYPANEL.md).
ENV API_HOST=wellness-api.p5beachclub.com.br
EXPOSE 80
