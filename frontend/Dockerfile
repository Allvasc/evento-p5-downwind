# Fica na raiz de frontend/ (não em apps/site/) de propósito: o EasyPanel usado neste
# projeto só aceita um único campo de path, que serve tanto de build context quanto de
# diretório onde ele procura um arquivo chamado exatamente "Dockerfile" — não dá pra
# apontar pra um Dockerfile aninhado com um build context diferente. Por isso o path
# aqui também precisa ser o build context certo, com o workspace pnpm inteiro visível
# (packages/shared é uma dependência real do app). No EasyPanel: Path = frontend.
FROM node:22-alpine AS build
RUN corepack enable && corepack prepare pnpm@9 --activate
WORKDIR /repo
COPY pnpm-workspace.yaml pnpm-lock.yaml ./
COPY packages ./packages
COPY apps ./apps
RUN pnpm install --frozen-lockfile
RUN pnpm --filter @p5wellness/site build

FROM nginx:1.27-alpine
COPY apps/site/nginx.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /repo/apps/site/dist /usr/share/nginx/html
# Domínio público da API (sem esquema) — só precisa mudar se o domínio real for
# diferente deste (ver deploy/EASYPANEL.md).
ENV API_HOST=downwindday-api.p5beachclub.com.br
EXPOSE 80
