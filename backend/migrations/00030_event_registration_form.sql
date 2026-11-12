-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- Inscrição por formulário (sem conta, sem pagamento) para eventos como o
-- CORE Demo Day: a pessoa preenche o formulário na própria página do evento e
-- a equipe acompanha as inscrições no Admin (aba Inscrições / CSV).
-- Eventos em modo 'tickets' (o padrão, inclusive o evento principal) continuam
-- vendendo ingressos pelo checkout, exatamente como antes.
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
ALTER TABLE events
  ADD COLUMN registration_mode text NOT NULL DEFAULT 'tickets' CHECK (registration_mode IN ('tickets','form')),
  ADD COLUMN form_capacity     integer CHECK (form_capacity IS NULL OR form_capacity > 0), -- NULL = sem limite
  ADD COLUMN waitlist_enabled  boolean NOT NULL DEFAULT true,
  ADD COLUMN rules_text        text,
  ADD COLUMN cta_label         text,
  ADD COLUMN meta_description  text;

CREATE TABLE event_registrations (
  id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id          uuid NOT NULL REFERENCES events(id),
  full_name         text NOT NULL,
  whatsapp          varchar(20) NOT NULL,
  email             citext NOT NULL,
  practice_level    text NOT NULL CHECK (practice_level IN ('iniciante','intermediario','avancado')),
  interest          text NOT NULL CHECK (interest IN ('kite','prancha','ambos')),
  weight_kg         integer CHECK (weight_kg IS NULL OR weight_kg BETWEEN 20 AND 250),
  status            text NOT NULL DEFAULT 'confirmed' CHECK (status IN ('confirmed','waitlist','cancelled')),
  -- LGPD: registro do aceite das regras e do consentimento, com o texto exibido.
  rules_accepted_at timestamptz NOT NULL,
  consent_at        timestamptz NOT NULL,
  consent_text      text NOT NULL,
  checked_in_at     timestamptz,
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now(),
  UNIQUE (event_id, email)
);
CREATE INDEX idx_event_registrations_event ON event_registrations(event_id, status);

-- Métrica de visitas da página do evento: só um contador por dia, sem IP nem
-- qualquer dado pessoal (LGPD: minimização).
CREATE TABLE event_page_views (
  event_id uuid NOT NULL REFERENCES events(id),
  day      date NOT NULL,
  views    integer NOT NULL DEFAULT 0,
  PRIMARY KEY (event_id, day)
);
-- +goose StatementEnd

-- +goose StatementBegin
-- CORE Demo Day — domingo, 15/11/2026, 08h às 14h, na P5 Kite House.
-- Inscrição gratuita pelo formulário da página /core-demo-day. Sem limite de vagas
-- até a organização definir o número (Admin → Eventos); lista de espera ligada.
INSERT INTO events (
  id, name, slug, is_default, active, registration_mode, waitlist_enabled,
  headline, subtitle, date_label, location_label, description, includes, schedule,
  rules_text, cta_label, meta_description, display_order
) VALUES (
  '00000000-0000-0000-0000-0000000000e2',
  'CORE Demo Day',
  'core-demo-day',
  false,
  true,
  'form',
  true,
  'Sinta a evolução. Teste a performance. Viva a experiência CORE.',
  'Demo Day de equipamentos CORE na P5 Kite House',
  'Domingo, 15 de novembro · 08h às 14h',
  'P5 Kite House · Av. Clóvis Arrais Maia, 4455, Praia do Futuro, Fortaleza/CE',
  E'No domingo, 15 de novembro, a P5 Kite House recebe o CORE Demo Day: uma manhã especial para conhecer e experimentar equipamentos CORE em condições reais de navegação. Descubra tecnologias, tire dúvidas e sinta a performance na água, em um encontro com a comunidade do kitesurf.\n\nVagas limitadas para testes. Faça sua inscrição nesta página e garanta sua vaga na programação!',
  E'Equipamentos e tecnologias CORE | Apresentação dos equipamentos e das tecnologias CORE.\nSessões de testes na água | Conforme as condições de vento e de segurança.\nOrientação da equipe | Orientações e troca de experiências com a equipe responsável.\nEstrutura P5 Kite House | Estrutura, gastronomia e ambiente da P5 Kite House.',
  E'08h · Recepção e credenciamento\n08h30 · Apresentação dos equipamentos\n09h às 13h30 · Sessões de testes organizadas\n13h30 às 14h · Encerramento',
  E'A confirmação da inscrição garante uma vaga para teste, mas não um modelo específico nem horário fixo.\nA participação na água depende de autonomia de navegação, disponibilidade dos equipamentos, condições meteorológicas e avaliação da equipe.\nO evento poderá ter sessões ajustadas ou suspensas por segurança.\nMenores de idade somente conforme política aprovada pela organização e com autorização do responsável.',
  'Garantir minha vaga',
  'CORE Demo Day na P5 Kite House | 15/11, das 08h às 14h. Conheça e teste equipamentos CORE. Inscreva-se no site e garanta sua vaga.',
  (SELECT COALESCE(MAX(display_order), 0) + 1 FROM events)
)
ON CONFLICT (slug) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM event_page_views    WHERE event_id = '00000000-0000-0000-0000-0000000000e2';
DELETE FROM event_registrations WHERE event_id = '00000000-0000-0000-0000-0000000000e2';
DELETE FROM events              WHERE id = '00000000-0000-0000-0000-0000000000e2';
DROP TABLE event_page_views;
DROP TABLE event_registrations;
ALTER TABLE events
  DROP COLUMN meta_description,
  DROP COLUMN cta_label,
  DROP COLUMN rules_text,
  DROP COLUMN waitlist_enabled,
  DROP COLUMN form_capacity,
  DROP COLUMN registration_mode;
-- +goose StatementEnd
