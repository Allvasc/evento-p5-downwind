# Deploy no EasyPanel

Estratégia: 3 serviços separados dentro do mesmo projeto EasyPanel (Postgres gerenciado
+ 2 Apps), cada um com seu próprio domínio e TLS automático pelo Traefik do EasyPanel.
Diferente do deploy alternativo em `docker-compose.prod.yml`/`Caddyfile` (pensado para
uma VPS "crua", sem gerenciador), aqui **não se usa Caddy** — o frontend serve seus
próprios estáticos via nginx e encaminha `/api/*` **para o domínio público da API**
(`downwindday-api.p5beachclub.com.br`, HTTPS) — não para um hostname interno da
plataforma. Evita depender de convenção de rede específica do EasyPanel (nome de
serviço interno, ordem de start etc.) — o mesmo Dockerfile funciona em qualquer PaaS
sem ajuste, desde que o domínio da API já esteja no ar.

O frontend (`frontend/apps/site`) é um único app Vue que serve tanto o site público
(landing page, checkout, portal do aluno) quanto o painel interno da equipe (`/admin`,
`/check-in`) — não existe mais um app/domínio separado para o admin. As rotas do
painel exigem login de equipe (`/acesso-admin`) e o nginx marca essas páginas com
`X-Robots-Tag: noindex, nofollow` para não aparecerem em buscadores.

**Sobre o campo de path no EasyPanel**: nesta instância ele só tem **um** campo de
path por app — não tem "build context" e "caminho do Dockerfile" separados. Esse path
vira o build context *e* o EasyPanel sempre procura um arquivo chamado exatamente
`Dockerfile` bem na raiz dele (nunca aninhado). Por isso os dois Dockerfiles deste
repo (`backend/Dockerfile` e `frontend/Dockerfile`) ficam direto na raiz das suas
respectivas pastas de build context — não tem como apontar pra um Dockerfile dentro
de uma subpasta com um path diferente pro contexto.

## 0. Antes de começar

Aponte os 2 domínios (A/AAAA) para o IP do servidor onde o EasyPanel roda:
`downwindday.p5beachclub.com.br`, `downwindday-api.p5beachclub.com.br`.

## 1. Banco de dados

Crie um serviço **Postgres** pelo template nativo do EasyPanel (não precisa ser via
Dockerfile). Anote o host interno, porta, usuário, senha e nome do banco que o
EasyPanel gerar — vai virar a `DATABASE_URL` do backend.

## 2. App do backend

- **Fonte**: este repositório Git.
- **Path**: `backend` (o Dockerfile já existe em `backend/Dockerfile`, builda `cmd/api` e `cmd/seed-admin`)
- **Porta do container**: `8090`
- **Domínio**: `downwindday-api.p5beachclub.com.br` (é a URL que vai no webhook da Asaas)
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
  SMTP_USER=downwindday@p5beachclub.com.br
  SMTP_PASS=<senha da caixa — nunca commitar>
  SMTP_FROM=P5 DownWind Day <downwindday@p5beachclub.com.br>
  ASAAS_API_KEY=<chave de produção — nunca a sandbox>
  ASAAS_BASE_URL=https://api.asaas.com/v3
  ASAAS_WEBHOOK_TOKEN=<o mesmo valor cadastrado no webhook da Asaas>
  PUBLIC_SITE_URL=https://downwindday.p5beachclub.com.br
  PUBLIC_ADMIN_URL=https://downwindday.p5beachclub.com.br
  ```
  `PUBLIC_SITE_URL` e `PUBLIC_ADMIN_URL` apontam para o mesmo domínio agora (site e
  admin são o mesmo app) — mantidos como duas variáveis só porque o backend ainda as
  lê separadamente (CORS e link nos e-mails).
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

Faça o deploy do backend (passo 2) **antes** do frontend abaixo — ele espera que
`downwindday-api.p5beachclub.com.br` já responda, já que faz proxy direto pra lá.

## 3. App do frontend (`frontend/apps/site`, servido a partir de `frontend/Dockerfile`)

- **Path**: `frontend` (o Dockerfile fica na raiz desta pasta de propósito — precisa
  enxergar o workspace pnpm inteiro, incluindo `packages/shared`, não só `apps/site`)
- **Porta do container**: `80`
- **Domínio**: `downwindday.p5beachclub.com.br`
- **Variáveis de ambiente**: nenhuma obrigatória — o Dockerfile já tem
  `API_HOST=downwindday-api.p5beachclub.com.br` como padrão, que é o domínio real. Só
  defina `API_HOST` explicitamente se for usar um domínio de API diferente.

## 4. Webhook da Asaas

Configure na conta de **produção** da Asaas (ver instruções detalhadas já trocadas
antes): URL `https://downwindday-api.p5beachclub.com.br/api/v1/webhooks/asaas`, token
igual ao `ASAAS_WEBHOOK_TOKEN` do app do backend, eventos `PAYMENT_CONFIRMED`,
`PAYMENT_RECEIVED`, `PAYMENT_OVERDUE`, `PAYMENT_DELETED`, `PAYMENT_REFUNDED`.

## 5. Depois do ar

- SPF/DKIM/DMARC do domínio (`docs/EMAIL_DNS_SETUP.md`) — sem isso os e-mails caem em spam.
- Backups do Postgres — confirme se o plano/instância do EasyPanel já inclui backup
  automático do banco; se não, configure `pg_dump` agendado separadamente.
- Revise `docs/LGPD.md` antes de operar com dados reais de clientes.
- Troque o catálogo de demonstração (Yoga/HYROX, seedado pela migration
  `00002_seed_demo_catalog.sql`) pelo ingresso real do P5 DownWind Day — dá pra editar
  tudo direto no painel admin (aba Produtos/Turmas, em `/admin`), sem precisar de
  migration nova.
