-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- Eventos simultâneos. Cada evento tem sua própria página (/<slug>), seus
-- produtos/preços, atividades/turmas, pedidos e vouchers — nada se mistura.
--
-- O evento que já está no ar vira o "evento padrão" (is_default): continua na
-- página "/" (Home.vue, conteúdo fixo no código) e em "/comprar". Tudo que já
-- existe é ligado a ele pelo DEFAULT das colunas novas, então o site atual não
-- muda nada. Os eventos novos ganham uma página genérica, editada pelo Admin.
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
CREATE TABLE events (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name           text NOT NULL,
  slug           text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
  is_default     boolean NOT NULL DEFAULT false,
  -- active = página publicada e ingressos à venda. O evento padrão nunca é desativado.
  active         boolean NOT NULL DEFAULT false,
  -- Conteúdo da página /<slug> (o evento padrão usa a Home.vue e ignora estes campos).
  headline       text,
  subtitle       text,
  date_label     text,
  location_label text,
  description    text,
  banner_url     text,
  includes       text, -- um item por linha ("Título | detalhe")
  schedule       text, -- um item por linha ("Horário · Título | detalhe")
  display_order  integer NOT NULL DEFAULT 0,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_events_single_default ON events (is_default) WHERE is_default;

INSERT INTO events (id, name, slug, is_default, active)
VALUES ('00000000-0000-0000-0000-0000000000e1', 'P5 DownWind Day', 'p5-downwind-day', true, true);
-- +goose StatementEnd

-- +goose StatementBegin
-- O DEFAULT liga ao evento padrão tudo o que já existe e qualquer código antigo que
-- ainda insira sem informar o evento. O Admin sempre informa o evento escolhido.
ALTER TABLE activities ADD COLUMN event_id uuid NOT NULL DEFAULT '00000000-0000-0000-0000-0000000000e1' REFERENCES events(id);
ALTER TABLE products   ADD COLUMN event_id uuid NOT NULL DEFAULT '00000000-0000-0000-0000-0000000000e1' REFERENCES events(id);
ALTER TABLE vouchers   ADD COLUMN event_id uuid NOT NULL DEFAULT '00000000-0000-0000-0000-0000000000e1' REFERENCES events(id);
ALTER TABLE orders     ADD COLUMN event_id uuid NOT NULL DEFAULT '00000000-0000-0000-0000-0000000000e1' REFERENCES events(id);

CREATE INDEX idx_activities_event ON activities(event_id);
CREATE INDEX idx_products_event   ON products(event_id);
CREATE INDEX idx_vouchers_event   ON vouchers(event_id);
CREATE INDEX idx_orders_event     ON orders(event_id);
-- +goose StatementEnd

-- +goose StatementBegin
-- Check-in: QR de um evento lido no terminal de outro evento.
ALTER TABLE validation_log DROP CONSTRAINT validation_log_result_check;
ALTER TABLE validation_log ADD CONSTRAINT validation_log_result_check
  CHECK (result IN ('success','already_used','expired','not_found','invalid_signature','override','wrong_sector','wrong_event'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM validation_log WHERE result = 'wrong_event';
ALTER TABLE validation_log DROP CONSTRAINT validation_log_result_check;
ALTER TABLE validation_log ADD CONSTRAINT validation_log_result_check
  CHECK (result IN ('success','already_used','expired','not_found','invalid_signature','override','wrong_sector'));

ALTER TABLE orders     DROP COLUMN event_id;
ALTER TABLE vouchers   DROP COLUMN event_id;
ALTER TABLE products   DROP COLUMN event_id;
ALTER TABLE activities DROP COLUMN event_id;
DROP TABLE events;
-- +goose StatementEnd
