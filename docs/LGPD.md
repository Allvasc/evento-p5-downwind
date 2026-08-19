# Checklist LGPD — antes de ir para produção com dados reais

## O que já está implementado

- **CPF criptografado em repouso** (`pgcrypto`/`pgp_sym_encrypt`) na tabela `students`, nunca armazenado em texto puro. Um hash determinístico (`cpf_hash`) permite busca/unicidade sem expor o valor.
- **Senhas com bcrypt + pepper** (segredo de aplicação somado antes do hash) para alunos e equipe.
- **HTTPS obrigatório** em produção (ver `deploy/Caddyfile` — TLS automático via Let's Encrypt).
- **Base legal**: execução de contrato (compra de experiências) para os dados de cadastro do aluno.
- **Minimização**: CPF é opcional no cadastro do aluno (campo de redundância para localizar em caso de erro, não obrigatório para a compra).
- **Anonimização por solicitação de exclusão** (art. 18): `POST /api/v1/admin/customers/:id/anonymize` (painel admin, aba Clientes) faz soft delete (`deleted_at`) preservando o histórico de pedidos por obrigação fiscal, mas apaga nome/e-mail/telefone/CPF do registro em `students`. Irreversível — a conta não pode mais entrar depois.

## Pendências antes de produção com dados reais
- [ ] **Política de privacidade** publicada no site (`/comprar`, `/entrar`) explicando quais dados são coletados, por quê, e por quanto tempo são retidos.
- [ ] **Registro de consentimento** no cadastro (checkbox de aceite dos termos, com timestamp).
- [ ] **Definir prazo de retenção** de dados de alunos inativos e configurar rotina de expurgo.
- [ ] **Revisar `QR_HMAC_SECRET`, `JWT_SECRET`, `PASSWORD_PEPPER`**: devem ser valores aleatórios fortes (32+ bytes), gerados uma vez e nunca commitados — ver `.env.example`.
- [ ] **Backups do Postgres**: automatizar (ex: `pg_dump` diário para storage externo) e criptografar os backups em repouso, já que contêm dados pessoais.
- [ ] **Log de acesso**: `validation_log` e `email_log` já registram quem fez o quê e quando — importante manter para auditoria, mas também sujeito a retenção/expurgo.
- [ ] Nomear um responsável (mesmo informal, dado o porte) para atender solicitações de titulares de dados.

## Dados sensíveis no sistema

| Dado | Onde fica | Proteção atual |
|---|---|---|
| CPF | `students.cpf_encrypted` | Criptografado (pgcrypto) |
| Senha | `students.password_hash`, `team_members.password_hash` | bcrypt + pepper |
| E-mail/telefone | `students` (texto simples) | Acesso restrito por auth; considerar criptografia se o volume de dados justificar |
| Token de sessão | JWT (client-side) | Assinado (HMAC), expira em 7 dias |
