-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;
-- +goose StatementEnd

-- ═══════════════════════════════════════════════════════════════════════════
-- Alunos (clientes) — cadastro com senha, autenticados para comprar
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
CREATE TABLE students (
  id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  full_name         text NOT NULL,
  cpf_encrypted     bytea,
  cpf_hash          char(64) UNIQUE,
  cpf_last4         varchar(4),
  email             citext NOT NULL UNIQUE,
  phone             varchar(20),
  password_hash     text NOT NULL,
  asaas_customer_id text,
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now(),
  deleted_at        timestamptz
);
-- +goose StatementEnd

-- ═══════════════════════════════════════════════════════════════════════════
-- Equipe (admin / staff) — contas criadas pelo admin, nunca autocadastro
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
CREATE TABLE team_members (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name          text NOT NULL,
  email         citext NOT NULL UNIQUE,
  phone         varchar(20),
  password_hash text NOT NULL,
  role          text NOT NULL CHECK (role IN ('admin','staff')),
  active        boolean NOT NULL DEFAULT true,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  last_login_at timestamptz
);
-- +goose StatementEnd

-- ═══════════════════════════════════════════════════════════════════════════
-- Recuperação de senha (alunos e equipe) — código curto, expira, uso único
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
CREATE TABLE password_reset_tokens (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_type  text NOT NULL CHECK (owner_type IN ('student','team_member')),
  owner_id    uuid NOT NULL,
  code_hash   text NOT NULL,
  expires_at  timestamptz NOT NULL,
  consumed_at timestamptz,
  attempts    integer NOT NULL DEFAULT 0,
  ip_address  inet,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_password_reset_owner ON password_reset_tokens(owner_type, owner_id);
-- +goose StatementEnd

-- ═══════════════════════════════════════════════════════════════════════════
-- Atividades (modalidades) — ex: Yoga, HYROX
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
CREATE TABLE activities (
  id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  title            text NOT NULL,
  slug             text NOT NULL UNIQUE,
  instructor       text,
  duration_minutes integer,
  description      text,
  image_url        text,
  active           boolean NOT NULL DEFAULT true,
  display_order    integer NOT NULL DEFAULT 0,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- ═══════════════════════════════════════════════════════════════════════════
-- Turmas (sessões agendadas de uma atividade, com vagas limitadas)
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
CREATE TABLE class_sessions (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  activity_id uuid NOT NULL REFERENCES activities(id),
  starts_at   timestamptz NOT NULL,
  ends_at     timestamptz NOT NULL,
  capacity    integer NOT NULL DEFAULT 50,
  status      text NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled','cancelled','completed')),
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_class_sessions_activity ON class_sessions(activity_id);
CREATE INDEX idx_class_sessions_starts_at ON class_sessions(starts_at);
-- +goose StatementEnd

-- ═══════════════════════════════════════════════════════════════════════════
-- Produtos (cards vendidos em /comprar) — aula avulsa, combo de aulas, combo+café
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
CREATE TABLE products (
  id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  title               text NOT NULL,
  slug                text NOT NULL UNIQUE,
  description         text,
  type                text NOT NULL CHECK (type IN ('class','combo')),
  includes_breakfast  boolean NOT NULL DEFAULT false,
  price_cents         integer NOT NULL,
  image_url           text,
  featured            boolean NOT NULL DEFAULT false,
  active              boolean NOT NULL DEFAULT true,
  display_order       integer NOT NULL DEFAULT 0,
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now()
);

-- Quais atividades cada produto dá direito a escolher/agendar (combo pode ter N)
CREATE TABLE product_activities (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  product_id  uuid NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  activity_id uuid NOT NULL REFERENCES activities(id),
  UNIQUE(product_id, activity_id)
);
-- +goose StatementEnd

-- ═══════════════════════════════════════════════════════════════════════════
-- Pedidos
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
CREATE TABLE orders (
  id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_number     text NOT NULL UNIQUE,
  student_id       uuid NOT NULL REFERENCES students(id),
  status           text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','paid','expired','failed','refunded','cancelled')),
  total_cents      integer NOT NULL,
  payment_method   text CHECK (payment_method IN ('pix','credit_card','boleto')),
  asaas_payment_id text,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now(),
  paid_at          timestamptz
);
CREATE INDEX idx_orders_student ON orders(student_id);
CREATE INDEX idx_orders_status ON orders(status);

CREATE TABLE order_items (
  id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id          uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  product_id        uuid NOT NULL REFERENCES products(id),
  activity_id       uuid REFERENCES activities(id),        -- NULL quando benefit_type='breakfast'
  class_session_id  uuid REFERENCES class_sessions(id),    -- turma escolhida no checkout, quando aplicável
  benefit_type      text NOT NULL CHECK (benefit_type IN ('class','breakfast')),
  unit_price_cents  integer NOT NULL
);
-- +goose StatementEnd

-- ═══════════════════════════════════════════════════════════════════════════
-- Entitlements (= ingressos/QR) — 1 por benefício do pedido, validado à parte
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
CREATE TABLE entitlements (
  id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_item_id    uuid NOT NULL REFERENCES order_items(id),
  student_id       uuid NOT NULL REFERENCES students(id),
  benefit_type     text NOT NULL CHECK (benefit_type IN ('class','breakfast')),
  class_session_id uuid REFERENCES class_sessions(id),
  qr_token         text NOT NULL UNIQUE,                    -- formato p5_<base62>
  status           text NOT NULL DEFAULT 'available' CHECK (status IN ('available','used','expired','cancelled')),
  valid_from       date NOT NULL,
  valid_until      date NOT NULL,
  used_at          timestamptz,
  used_by          uuid REFERENCES team_members(id),
  issued_at        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_entitlements_student ON entitlements(student_id);
CREATE INDEX idx_entitlements_status ON entitlements(status);
CREATE INDEX idx_entitlements_qr_token ON entitlements(qr_token);
-- +goose StatementEnd

-- ═══════════════════════════════════════════════════════════════════════════
-- Pagamentos (espelho Asaas) + idempotência de webhook
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
CREATE TABLE payments (
  id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id          uuid NOT NULL REFERENCES orders(id),
  asaas_payment_id  text NOT NULL UNIQUE,
  asaas_customer_id text NOT NULL,
  status            text NOT NULL,
  billing_type      text NOT NULL,
  amount_cents      integer NOT NULL,
  raw_payload       jsonb,
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE webhook_events (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  provider     text NOT NULL DEFAULT 'asaas',
  dedupe_hash  char(64) NOT NULL,
  event_type   text,
  payload      jsonb,
  received_at  timestamptz NOT NULL DEFAULT now(),
  processed_at timestamptz,
  UNIQUE(provider, dedupe_hash)
);
-- +goose StatementEnd

-- ═══════════════════════════════════════════════════════════════════════════
-- E-mail e auditoria de validação (check-in)
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
CREATE TABLE email_log (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  to_email      citext NOT NULL,
  subject       text NOT NULL,
  template      text NOT NULL,
  order_id      uuid REFERENCES orders(id),
  status        text NOT NULL DEFAULT 'queued' CHECK (status IN ('queued','sent','failed')),
  attempts      integer NOT NULL DEFAULT 0,
  error_message text,
  created_at    timestamptz NOT NULL DEFAULT now(),
  sent_at       timestamptz
);

CREATE TABLE validation_log (
  id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  entitlement_id    uuid REFERENCES entitlements(id),
  scanned_by        uuid NOT NULL REFERENCES team_members(id),
  result            text NOT NULL CHECK (result IN ('success','already_used','expired','not_found','invalid_signature','override')),
  client_scan_id    uuid NOT NULL,
  device_scanned_at timestamptz NOT NULL,
  synced_at         timestamptz NOT NULL DEFAULT now(),
  UNIQUE(client_scan_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS validation_log;
DROP TABLE IF EXISTS email_log;
DROP TABLE IF EXISTS webhook_events;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS entitlements;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS product_activities;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS class_sessions;
DROP TABLE IF EXISTS activities;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS students;
-- +goose StatementEnd
