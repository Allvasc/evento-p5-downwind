# Configuração de DNS para e-mail transacional

Sem SPF, DKIM e DMARC configurados no domínio, os e-mails de confirmação de compra (com os QR Codes) e de recuperação de senha **vão cair em spam** com alta probabilidade — isso não é opcional.

## Hostgator (produção)

Conta de e-mail já ativa no cPanel: `wellness@p5beachclub.com.br`.

1. Configure no `.env` de produção (raiz do repo, nunca commitado — ver `deploy/.env.production.example`):
   ```
   SMTP_HOST=mail.p5beachclub.com.br
   SMTP_PORT=465                       # SSL implícito — o dialer detecta automaticamente pela porta
   SMTP_USER=wellness@p5beachclub.com.br
   SMTP_PASS=<senha da caixa de e-mail — não versionar>
   SMTP_FROM="P5 Wellness Club <wellness@p5beachclub.com.br>"
   ```
2. No painel de DNS do domínio `p5beachclub.com.br` (Zone Editor do cPanel, ou onde o domínio está registrado):
   - **SPF** (registro TXT na raiz do domínio): autoriza os servidores da Hostgator a enviar em nome do domínio.
     ```
     v=spf1 +a +mx include:hostgator.com.br ~all
     ```
     (o valor exato do `include` pode variar — o cPanel geralmente mostra o SPF recomendado em *E-mail > Autenticação*).
   - **DKIM**: no cPanel, vá em *E-mail > Autenticação por E-mail* e ative o DKIM — ele gera automaticamente o registro TXT (`default._domainkey.p5beachclub.com.br`) para você publicar no DNS.
   - **DMARC** (registro TXT em `_dmarc.p5beachclub.com.br`):
     ```
     v=DMARC1; p=quarantine; rua=mailto:wellness@p5beachclub.com.br; pct=100
     ```
     Comece com `p=quarantine` (mais seguro que `p=none`, menos agressivo que `p=reject`) e evolua para `p=reject` depois de confirmar que os envios legítimos não estão sendo bloqueados.
3. Após publicar os três registros, valide em [mxtoolbox.com/SuperTool.aspx](https://mxtoolbox.com) (SPF, DKIM, DMARC) antes de considerar produção pronta.

**Segurança**: a senha real da caixa `wellness@p5beachclub.com.br` fica apenas no `.env` de produção na VPS (fora do git) e, para testes locais, em `backend/.env` (também fora do git — confira `git check-ignore backend/.env` antes de qualquer commit). Se ela já foi compartilhada em algum canal não seguro, troque-a no cPanel antes de ir ao ar.

## Gmail (apenas desenvolvimento/testes)

```
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=seuemail@gmail.com
SMTP_PASS=<senha de app, não a senha normal da conta>
```

Gere uma "senha de app" em [myaccount.google.com/apppasswords](https://myaccount.google.com/apppasswords) (exige verificação em duas etapas ativada). **Não usar em produção**: limite de ~500 envios/dia e maior chance de outros provedores marcarem como spam por ser uma conta pessoal enviando e-mail automatizado.

## Sem SMTP configurado

Se `SMTP_HOST` estiver vazio, o backend usa um sender de desenvolvimento que só grava em log e na tabela `email_log` — nenhum e-mail real é enviado. Isso é o comportamento padrão em `APP_ENV=development` até o SMTP ser configurado.
