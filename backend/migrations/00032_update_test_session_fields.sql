-- +goose Up
-- +goose StatementBegin
UPDATE events
   SET name             = 'Test Session',
       slug             = 'test-session',
       subtitle         = 'Test Session de equipamentos CORE na P5 Kite House',
       description      = E'No domingo, 15 de novembro, a P5 Kite House recebe o Test Session: uma manhã especial para conhecer e experimentar equipamentos CORE em condições reais de navegação. Descubra tecnologias, tire dúvidas e sinta a performance na água, em um encontro com a comunidade do kitesurf.\n\nVagas limitadas para testes. Faça sua inscrição nesta página e garanta sua vaga na programação!',
       meta_description = 'Test Session na P5 Kite House | 15/11, das 08h às 14h. Conheça e teste equipamentos CORE. Inscreva-se no site e garanta sua vaga.',
       updated_at       = now()
 WHERE id = '00000000-0000-0000-0000-0000000000e2';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE events
   SET slug             = 'core-demo-day',
       subtitle         = 'Demo Day de equipamentos CORE na P5 Kite House',
       description      = E'No domingo, 15 de novembro, a P5 Kite House recebe o CORE Demo Day: uma manhã especial para conhecer e experimentar equipamentos CORE em condições reais de navegação. Descubra tecnologias, tire dúvidas e sinta a performance na água, em um encontro com a comunidade do kitesurf.\n\nVagas limitadas para testes. Faça sua inscrição nesta página e garanta sua vaga na programação!',
       meta_description = 'CORE Demo Day na P5 Kite House | 15/11, das 08h às 14h. Conheça e teste equipamentos CORE. Inscreva-se no site e garanta sua vaga.',
       updated_at       = now()
 WHERE id = '00000000-0000-0000-0000-0000000000e2';
-- +goose StatementEnd

