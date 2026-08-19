-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- Setores/empresas parceiras (ex: P5 = barraca/café, AYO = fitness).
-- Cada entitlement (QR) pertence a exatamente um setor, e cada atividade
-- pertence a exatamente um setor. Um pedido com Yoga + HYROX + Café da Manhã
-- gera 2 QR codes: um da AYO (cobre Yoga e HYROX juntos) e um da P5 (café).
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
CREATE TABLE vendors (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name       text NOT NULL,
  slug       text NOT NULL UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO vendors (id, name, slug) VALUES
  ('00000000-0000-0000-0000-0000000000f5', 'P5 Kite House', 'p5'),
  ('00000000-0000-0000-0000-0000000000a1', 'AYO Fitness',   'ayo');
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE activities ADD COLUMN vendor_id uuid REFERENCES vendors(id);
UPDATE activities SET vendor_id = '00000000-0000-0000-0000-0000000000a1'; -- Yoga/HYROX seed → AYO
ALTER TABLE activities ALTER COLUMN vendor_id SET NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
-- NULL = acesso irrestrito (admin da P5, enxerga/valida todos os setores).
-- Preenchido = funcionário restrito a validar apenas o QR do seu próprio setor.
ALTER TABLE team_members ADD COLUMN vendor_id uuid REFERENCES vendors(id);
-- +goose StatementEnd

-- +goose StatementBegin
-- Entitlements deixam de ser 1:1 com order_items (1 por atividade) e passam a
-- ser 1:1 com (pedido, setor) — cada um pode cobrir várias atividades do
-- mesmo setor dentro do pedido.
CREATE TABLE entitlement_items (
  entitlement_id uuid NOT NULL,
  order_item_id  uuid NOT NULL REFERENCES order_items(id),
  PRIMARY KEY (entitlement_id, order_item_id)
);

ALTER TABLE entitlements ADD COLUMN order_id uuid REFERENCES orders(id);
ALTER TABLE entitlements ADD COLUMN vendor_id uuid REFERENCES vendors(id);

-- Nenhuma linha real de produção ainda depende do formato antigo (fase de
-- desenvolvimento) — zeramos os tickets de teste em vez de migrar dado a dado.
DELETE FROM validation_log;
DELETE FROM entitlements;

ALTER TABLE entitlements DROP COLUMN order_item_id;
ALTER TABLE entitlements ALTER COLUMN order_id SET NOT NULL;
ALTER TABLE entitlements ALTER COLUMN vendor_id SET NOT NULL;

ALTER TABLE entitlement_items
  ADD CONSTRAINT entitlement_items_entitlement_fk
  FOREIGN KEY (entitlement_id) REFERENCES entitlements(id) ON DELETE CASCADE;

CREATE INDEX idx_entitlements_order ON entitlements(order_id);
CREATE INDEX idx_entitlements_vendor ON entitlements(vendor_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE entitlements ADD COLUMN order_item_id uuid REFERENCES order_items(id);
DELETE FROM validation_log;
DELETE FROM entitlements;
ALTER TABLE entitlements ALTER COLUMN order_item_id SET NOT NULL;
ALTER TABLE entitlements DROP COLUMN order_id;
ALTER TABLE entitlements DROP COLUMN vendor_id;
DROP TABLE entitlement_items;
ALTER TABLE team_members DROP COLUMN vendor_id;
ALTER TABLE activities DROP COLUMN vendor_id;
DROP TABLE vendors;
-- +goose StatementEnd
