-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- Remove por completo o catálogo de demonstração (Yoga/HYROX + seus 3
-- produtos, seed 00002). A migration 00009 só o havia desativado
-- (active = false) por precaução; nesta fase de desenvolvimento nenhum
-- pedido real depende dele, então apagamos também os pedidos de teste que
-- o referenciavam — mesmo critério já usado em 00003.
--
-- As turmas (datas) do P5 DownWind Day NÃO são seedadas aqui: a equipe as
-- cadastra e gerencia pelo painel Admin → Turmas. Sem pelo menos uma turma
-- futura com vaga, o checkout não exibe o passo "Selecione sua data" nem a
-- landing o selo "próxima data".
--
-- O setor "AYO Fitness" (vendors) é mantido de propósito: não aparece em
-- lugar nenhum sem atividades ativas e a suíte de testes o usa como segundo
-- setor para cobrir a validação de QR por setor.
-- ═══════════════════════════════════════════════════════════════════════════

-- +goose StatementBegin
-- ── 1a. Apaga os pedidos de teste presos ao catálogo de demonstração ──────────
-- order_items referencia product_id/activity_id/class_session_id sem ON DELETE,
-- então a "cascata" abaixo é feita na mão, das folhas para a raiz.
CREATE TEMP TABLE _demo_orders ON COMMIT DROP AS
  SELECT DISTINCT oi.order_id
  FROM order_items oi
  WHERE oi.product_id  IN ('00000000-0000-0000-0000-000000000101',
                           '00000000-0000-0000-0000-000000000102',
                           '00000000-0000-0000-0000-000000000103')
     OR oi.activity_id IN ('00000000-0000-0000-0000-000000000001',
                           '00000000-0000-0000-0000-000000000002');

DELETE FROM validation_log
 WHERE entitlement_id IN (SELECT id FROM entitlements WHERE order_id IN (SELECT order_id FROM _demo_orders));
DELETE FROM entitlement_items
 WHERE entitlement_id IN (SELECT id FROM entitlements WHERE order_id IN (SELECT order_id FROM _demo_orders));
DELETE FROM entitlements     WHERE order_id IN (SELECT order_id FROM _demo_orders);
DELETE FROM order_reschedules WHERE order_id IN (SELECT order_id FROM _demo_orders);
UPDATE vouchers SET order_id = NULL, redeemed_by = NULL, redeemed_at = NULL, status = 'available'
 WHERE order_id IN (SELECT order_id FROM _demo_orders);
DELETE FROM email_log        WHERE order_id IN (SELECT order_id FROM _demo_orders);
DELETE FROM payments         WHERE order_id IN (SELECT order_id FROM _demo_orders);
DELETE FROM order_items      WHERE order_id IN (SELECT order_id FROM _demo_orders);
DELETE FROM orders           WHERE id       IN (SELECT order_id FROM _demo_orders);
-- +goose StatementEnd

-- +goose StatementBegin
-- ── 1b. Apaga o catálogo de demonstração em si ───────────────────────────────
DELETE FROM class_sessions WHERE activity_id IN ('00000000-0000-0000-0000-000000000001',
                                                 '00000000-0000-0000-0000-000000000002');
DELETE FROM product_activities WHERE product_id IN ('00000000-0000-0000-0000-000000000101',
                                                    '00000000-0000-0000-0000-000000000102',
                                                    '00000000-0000-0000-0000-000000000103');
DELETE FROM products   WHERE id IN ('00000000-0000-0000-0000-000000000101',
                                    '00000000-0000-0000-0000-000000000102',
                                    '00000000-0000-0000-0000-000000000103');
DELETE FROM activities WHERE id IN ('00000000-0000-0000-0000-000000000001',
                                    '00000000-0000-0000-0000-000000000002');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Recria o catálogo de demonstração (espelho do seed 00002), já desativado,
-- para manter a reversibilidade sem reabrir Yoga/HYROX na loja.
INSERT INTO activities (id, title, slug, instructor, duration_minutes, description, display_order, active, vendor_id) VALUES
  ('00000000-0000-0000-0000-000000000001', 'Yoga',  'yoga',  'Equipe AYO', 60, 'Práticas guiadas para todos os níveis, em uma atmosfera acolhedora.', 1, false, '00000000-0000-0000-0000-0000000000a1'),
  ('00000000-0000-0000-0000-000000000002', 'HYROX', 'hyrox', 'Equipe AYO', 60, 'Yoga e HYROX para acordar o corpo com intenção, presença e energia.', 2, false, '00000000-0000-0000-0000-0000000000a1');

INSERT INTO products (id, title, slug, description, type, includes_breakfast, price_cents, featured, active, display_order) VALUES
  ('00000000-0000-0000-0000-000000000101', 'Aula individual', 'aula-individual', 'Yoga ou HYROX — uma prática, um novo ritmo.', 'class', false, 2500, false, false, 1),
  ('00000000-0000-0000-0000-000000000102', 'Aulas', 'aulas-yoga-hyrox', 'Yoga + HYROX — movimento em dobro para viver a manhã.', 'combo', false, 4000, false, false, 2),
  ('00000000-0000-0000-0000-000000000103', 'Aulas + Café', 'aulas-cafe-manha', 'Yoga + HYROX + Café da Manhã — a experiência completa, da energia à mesa.', 'combo', true, 9000, false, false, 3);

INSERT INTO product_activities (product_id, activity_id) VALUES
  ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000001'),
  ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000002'),
  ('00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000001'),
  ('00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000002'),
  ('00000000-0000-0000-0000-000000000103', '00000000-0000-0000-0000-000000000001'),
  ('00000000-0000-0000-0000-000000000103', '00000000-0000-0000-0000-000000000002');
-- +goose StatementEnd
