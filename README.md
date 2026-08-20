# P5 DownWind Day

Sistema do P5 DownWind Day (P5 Kite House): landing page, ingresso único, checkout via
Asaas, emissão de QR Code, painel administrativo e terminal de check-in.

- **Backend**: Go (chi, pgx, JWT, bcrypt) — `backend/`
- **Frontend**: Vue 3 + Vite + Tailwind — um único app em `frontend/apps/site`, que serve
  tanto o site público (landing + checkout + área do aluno) quanto o painel interno da
  equipe (`/acesso-admin`, `/admin`, `/check-in`)
- **Banco**: PostgreSQL

## Rodando em desenvolvimento

```bash
# 1. Banco de dados
docker compose -f deploy/docker-compose.dev.yml up -d

# 2. Backend
cp .env.example backend/.env   # ajuste DATABASE_URL se mudar a porta do Postgres
cd backend
go run ./cmd/api   # aplica as migrations pendentes sozinho ao subir
# cria o primeiro administrador (uma vez):
go run ./cmd/seed-admin -name "Seu Nome" -email admin@p5kitehouse.dev -password "senha-forte"

# 3. Frontend
cd frontend
pnpm install
pnpm --filter @p5wellness/site dev     # http://localhost:5183
```

Sem `ASAAS_API_KEY` configurada, o checkout usa um cliente Asaas fake — suficiente para testar o fluxo completo de compra, e-mail e QR Code. Use `POST /api/v1/dev/simulate-payment/:orderId` (só existe fora de produção) para simular a confirmação do pagamento sem esperar um webhook real.

## Testes

```bash
cd backend
go test ./...
```

Os testes de domínio (validador de CPF) rodam sempre. Os testes de integração em `internal/repository/postgres` (concorrência na validação de QR, race de capacidade de turma no checkout) precisam de um Postgres real — são pulados automaticamente se `DATABASE_URL` não estiver configurada (via `backend/.env` ou variável de ambiente).

## Funcionalidades do painel admin

Acessível em `/admin` (login em `/acesso-admin`), além do CRUD de produtos/turmas/equipe e do dashboard:

- **Editar** produtos e turmas já criados (não só ativar/desativar) — turmas não aceitam capacidade menor que o já reservado.
- **Clientes**: busca por nome, e-mail, telefone ou últimos 4 dígitos do CPF (dado de redundância coletado no cadastro) — resolve o "paguei mas não achei o e-mail" sem acesso ao banco. Inclui **anonimização** (LGPD art. 18): apaga nome/e-mail/telefone/CPF mantendo o histórico de pedidos para fins fiscais.
- **Pedidos**: lista com filtro por status/busca, detalhe com itens e status de cada QR emitido, **reenvio de e-mail de confirmação** e **estorno** (reverte o pagamento na Asaas e cancela os benefícios ainda não utilizados).
- **Check-in offline** (`/check-in`): se a conexão cair no meio de um scan, o benefício é salvo localmente (IndexedDB) e sincronizado automaticamente assim que a conexão voltar — nenhum check-in se perde. Indicador de pendências e botão de sincronização manual no terminal.
- **Recuperação de senha da equipe**: `/acesso-admin` tem "Esqueci minha senha" para admin/staff, sem depender de acesso ao servidor (`cmd/seed-admin`).

O aluno também pode, em `/perfil`: adicionar/corrigir o CPF depois do cadastro (necessário para pagar) e reenviar o e-mail de um pedido pago.

## Estrutura

```
backend/         API Go
frontend/
  apps/site/     Site público (landing + checkout + área do aluno) e painel da equipe
                 (/acesso-admin, /admin, /check-in) — um único app
  packages/shared/  Tipos e cliente HTTP compartilhados
deploy/          Docker Compose (dev/prod) + Caddyfile
docs/            SPF/DKIM/DMARC, checklist LGPD
```

## Deploy em produção

Domínios: **downwindday.p5beachclub.com.br** (site + painel da equipe),
**downwindday-api.p5beachclub.com.br** (API — usado como URL do webhook da Asaas).

**No EasyPanel**: ver `deploy/EASYPANEL.md` — 3 serviços separados (Postgres gerenciado + 2 Apps via Dockerfile, cada um com domínio/TLS próprio pelo Traefik do EasyPanel).

**Em VPS "crua" (sem gerenciador)**: `deploy/docker-compose.prod.yml` + `deploy/Caddyfile` (Caddy assume TLS/proxy das portas 80/443 e serve os estáticos do frontend). Resumo:

0. Aponte os 2 domínios (A/AAAA) para o IP da VPS antes de subir — o Caddy só emite HTTPS automático se a DNS já resolver.
1. `cp deploy/.env.production.example .env` (raiz do repo) e preencha os segredos — gerar com `openssl rand -hex 32` para `JWT_SECRET`, `QR_HMAC_SECRET`, `PASSWORD_PEPPER`. **Nunca reutilize os valores de dev, nem a chave sandbox da Asaas.**
2. `pnpm --filter @p5wellness/site build`
3. `docker compose -f deploy/docker-compose.prod.yml up -d --build`
4. As migrations rodam sozinhas quando o `backend` sobe (embutidas no binário). Rode `docker compose -f deploy/docker-compose.prod.yml exec backend ./seed-admin -name "Seu Nome" -email admin@p5beachclub.com.br -password "senha-forte"` uma vez para criar o primeiro administrador.
5. Configure o webhook na conta de **produção** da Asaas (não confundir com sandbox) apontando para `https://downwindday-api.p5beachclub.com.br/api/v1/webhooks/asaas`, com o mesmo `ASAAS_WEBHOOK_TOKEN` do `.env`. Eventos mínimos: `PAYMENT_CONFIRMED`, `PAYMENT_RECEIVED`, `PAYMENT_OVERDUE`, `PAYMENT_DELETED`, `PAYMENT_REFUNDED`.
6. Confirme que `APP_ENV=production` está de fato ativo — isso desliga sozinho o endpoint `/api/v1/dev/simulate-payment` (só existe fora de produção).
7. Configure SPF/DKIM/DMARC no domínio antes de habilitar SMTP real — ver `docs/EMAIL_DNS_SETUP.md`.
8. Revise `docs/LGPD.md` antes de operar com dados reais de clientes.
