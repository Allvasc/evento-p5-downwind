# Deploy no EasyPanel

Estratégia: 4 serviços separados dentro do mesmo projeto EasyPanel (Postgres gerenciado
+ 3 Apps), cada um com seu próprio domínio e TLS automático pelo Traefik do EasyPanel.
Diferente do deploy alternativo em `docker-compose.prod.yml`/`Caddyfile` (pensado para
uma VPS "crua", sem gerenciador), aqui **não se usa Caddy** — cada frontend serve seus
próprios estáticos via nginx e encaminha `/api/*` **para o domínio público da API**
(`wellness-api.p5beachclub.com.br`, HTTPS) — não para um hostname interno da
plataforma. Evita depender de convenção de rede específica do EasyPanel (nome de
serviço interno, ordem de start etc.) — o mesmo Dockerfile funciona em qualquer PaaS
sem ajuste, desde que o domínio da API já esteja no ar.

## 0. Antes de começar

Aponte os 3 domínios (A/AAAA) para o IP do servidor onde o EasyPanel roda:
`wellness.p5beachclub.com.br`, `wellness-admin.p5beachclub.com.br`,
`wellness-api.p5beachclub.com.br`.

## 1. Banco de dados

Crie um serviço **Postgres** pelo template nativo do EasyPanel (não precisa ser via
Dockerfile). Anote o host interno, porta, usuário, senha e nome do banco que o
EasyPanel gerar — vai virar a `DATABASE_URL` do backend.

## 2. App do backend

- **Fonte**: este repositório Git.
- **Build Context**: `backend`
- **Dockerfile Path**: `backend/Dockerfile` (já existe, builda `cmd/api` e `cmd/seed-admin`)
- **Porta do container**: `8090`
- **Domínio**: `wellness-api.p5beachclub.com.br` (é a URL que vai no webhook da Asaas)
- **Variáveis de ambiente** (mesmas de `deploy/.env.production.example`, sem o prefixo
  `POSTGRES_*` que era só para o container do compose):
  ```
  APP_ENV=production
  PORT=8090
  DATABASE_URL=postgres://<user>:<pass>@<host-interno-do-postgres>:5432/<db>?sslmode=disable
  JWT_SECRET=<openssl rand -hex 32>
  QR_HMAC_SECRET=<openssl rand -hex 32>
  PASSWORD_PEPPER=<openssl rand -hex 32>
  SMTP_HOST=mail.p5beachclub.com.br
  SMTP_PORT=465
  SMTP_USER=wellness@p5beachclub.com.br
  SMTP_PASS=<senha da caixa — nunca commitar>
  SMTP_FROM=P5 Wellness Club <wellness@p5beachclub.com.br>
  ASAAS_API_KEY=<chave de produção — nunca a sandbox>
  ASAAS_BASE_URL=https://api.asaas.com/v3
  ASAAS_WEBHOOK_TOKEN=<o mesmo valor cadastrado no webhook da Asaas>
  PUBLIC_SITE_URL=https://wellness.p5beachclub.com.br
  PUBLIC_ADMIN_URL=https://wellness-admin.p5beachclub.com.br
  ```
- **As migrations rodam sozinhas** — o binário `api` aplica todas as migrations
  pendentes (embutidas nele em tempo de compilação) assim que sobe, antes de aceitar
  requisições. Não precisa rodar `goose` manualmente nem ter acesso externo ao banco.
  Confira nos logs do app: `migrations applied` logo no início.
- **Crie o primeiro administrador** usando o Terminal do próprio app do backend no
  EasyPanel (ele já tem a `DATABASE_URL` correta como variável de ambiente do
  container, então não precisa de acesso externo ao Postgres):
  ```bash
  ./seed-admin -name "Seu Nome" -email admin@p5beachclub.com.br -password "senha-forte"
  ```

Faça o deploy do backend (passo 2) **antes** dos dois frontends abaixo — eles esperam
que `wellness-api.p5beachclub.com.br` já responda, já que fazem proxy direto pra lá.

## 3. App do site (`frontend/apps/site`)

- **Build Context**: `frontend` (não `frontend/apps/site` — o Dockerfile precisa
  enxergar o workspace pnpm inteiro, incluindo `packages/shared`)
- **Dockerfile Path**: `apps/site/Dockerfile`
- **Porta do container**: `80`
- **Domínio**: `wellness.p5beachclub.com.br`
- **Variáveis de ambiente**: nenhuma obrigatória — o Dockerfile já tem
  `API_HOST=wellness-api.p5beachclub.com.br` como padrão, que é o domínio real. Só
  defina `API_HOST` explicitamente se for usar um domínio de API diferente.

## 4. App do admin (`frontend/apps/admin`)

Igual ao site, trocando:
- **Dockerfile Path**: `apps/admin/Dockerfile`
- **Domínio**: `wellness-admin.p5beachclub.com.br`

## 5. Webhook da Asaas

Configure na conta de **produção** da Asaas (ver instruções detalhadas já trocadas
antes): URL `https://wellness-api.p5beachclub.com.br/api/v1/webhooks/asaas`, token
igual ao `ASAAS_WEBHOOK_TOKEN` do app do backend, eventos `PAYMENT_CONFIRMED`,
`PAYMENT_RECEIVED`, `PAYMENT_OVERDUE`, `PAYMENT_DELETED`, `PAYMENT_REFUNDED`.

## 6. Depois do ar

- SPF/DKIM/DMARC do domínio (`docs/EMAIL_DNS_SETUP.md`) — sem isso os e-mails caem em spam.
- Backups do Postgres — confirme se o plano/instância do EasyPanel já inclui backup
  automático do banco; se não, configure `pg_dump` agendado separadamente.
- Revise `docs/LGPD.md` antes de operar com dados reais de clientes.
- Troque o catálogo de demonstração (Yoga/HYROX, seedado pela migration
  `00002_seed_demo_catalog.sql`) pelas atividades/produtos/turmas reais — dá pra editar
  tudo direto no painel admin (aba Atividades/Produtos/Turmas), sem precisar de migration nova.
