-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- Vincula a edição de 28/11 ao Sertões Kitesurf 2026 (Categoria Experience).
-- Também corrige a descrição: o pacote cobre a logística/transporte do
-- equipamento de kite de cada participante, não o equipamento em si.
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
UPDATE products
   SET title = 'P5 DownWind Day — Sertões Kitesurf Experience',
       description = 'Categoria Experience do Sertões Kitesurf 2026: Ponta Grossa (Icapuí) → Jaguaribe Lodge (Fortim), ~40 km. Inclui a inscrição oficial com kit participante (lycra, camiseta, boné e sacochila), café da manhã na P5, ônibus executivo ida e volta, logística para o seu equipamento de kite, equipe de acompanhamento com hidratação e snacks e confraternização no retorno.',
       updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000307';

UPDATE activities
   SET title = 'Sertões Kitesurf Experience · Icapuí → Fortim',
       description = 'Categoria Experience do Sertões Kitesurf 2026: ônibus da P5 até a Praia de Ponta Grossa (Icapuí) e downwind de aproximadamente 40 km até o Jaguaribe Lodge, em Fortim, com percurso acompanhado por guias exclusivos.',
       updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000205';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE products
   SET title = 'P5 DownWind Day — Sertões Experience',
       description = 'Icapuí → Fortim (~40 km). Inclui inscrição Sertões Experience, café da manhã na P5, kit do evento, ônibus executivo ida e volta, equipe de acompanhamento com hidratação e snacks, equipamentos de kite e entretenimento.',
       updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000307';

UPDATE activities
   SET title = 'DownWind Icapuí → Fortim',
       description = 'Sertões Experience: ônibus da P5 até Ponta Grossa (Icapuí) e downwind de aproximadamente 40 km até Fortim, com equipe de acompanhamento.',
       updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000205';
-- +goose StatementEnd
