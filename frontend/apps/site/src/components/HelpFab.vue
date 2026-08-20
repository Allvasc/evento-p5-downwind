<script setup lang="ts">
import { ref } from "vue";
import { HelpCircle, X, ChevronDown, Mail, QrCode, KeyRound, Backpack, CreditCard, CalendarClock } from "lucide-vue-next";

const open = ref(false);
const openQuestion = ref<number | null>(0);

interface FaqItem {
  icon: typeof HelpCircle;
  question: string;
  answer: string[];
}

// Grounded in how the app actually behaves (see Comprar.vue checkout, Perfil.vue resend,
// Entrar.vue password recovery) — every answer here should stay true to a real feature,
// not a policy nobody confirmed.
const faqs: FaqItem[] = [
  {
    icon: Mail,
    question: "Não recebi o e-mail de confirmação. E agora?",
    answer: [
      "O e-mail com o QR Code só é enviado depois que o pagamento é confirmado — pelo Pix isso costuma ser quase na hora, mas pode levar alguns minutos.",
      "Confira a caixa de spam/lixo eletrônico e, no Gmail, a aba \"Promoções\".",
      "Você pode reenviar sozinho: acesse Perfil → Meus pedidos → \"Reenviar e-mail\".",
      "Mesmo sem o e-mail em mãos, seu ingresso já fica disponível no Portal do Aluno.",
    ],
  },
  {
    icon: QrCode,
    question: "Comprei, e agora? Como recebo meu ingresso?",
    answer: [
      "Assim que o pagamento é confirmado, você recebe um e-mail com o QR Code de cada benefício da sua compra.",
      "Você também pode ver seus ingressos a qualquer momento pelo Portal do Aluno — faça login com o mesmo e-mail e senha que você cadastrou na hora da compra.",
      "Apresente o QR Code na entrada. Em combos (ex: aula + café da manhã), cada benefício é validado separadamente.",
    ],
  },
  {
    icon: KeyRound,
    question: "Esqueci minha senha do Portal do Aluno",
    answer: ["Na tela de login, clique em \"Esqueci minha senha\" e siga as instruções enviadas por e-mail."],
  },
  {
    icon: Backpack,
    question: "O que eu levo para o DownWind Day?",
    answer: [
      "Seu próprio equipamento de kite (kite, barra, trapézio e prancha) — o ingresso cobre estrutura e apoio, não o material.",
      "Protetor solar, roupa de neoprene ou lycra e água.",
      "O compressor para encher o kite fica disponível no ponto de saída.",
    ],
  },
  {
    icon: CreditCard,
    question: "Paguei e o status ainda mostra \"aguardando pagamento\"",
    answer: [
      "O pagamento é feito via Pix, processado pela Asaas — a confirmação costuma ser quase instantânea, mas pode levar alguns minutos em horários de instabilidade.",
      "Se passar de 30 minutos e o status não mudar, fale com a gente.",
    ],
  },
  {
    icon: CalendarClock,
    question: "Não vou conseguir comparecer. Posso remarcar?",
    answer: ["Sim — fale com a equipe P5 informando o número do seu pedido; a remarcação é feita manualmente pela equipe."],
  },
];

function toggle(i: number) {
  openQuestion.value = openQuestion.value === i ? null : i;
}
</script>

<template>
  <button
    class="fixed bottom-6 left-6 z-40 flex h-12 w-12 items-center justify-center rounded-full bg-ink text-white shadow-lg transition-transform hover:scale-105"
    aria-label="Dúvidas frequentes"
    @click="open = true"
  >
    <HelpCircle :size="22" />
  </button>

  <div v-if="open" class="fixed inset-0 z-50 flex items-end justify-center bg-ink/40 p-4 sm:items-center" @click.self="open = false">
    <div class="max-h-[85vh] w-full max-w-lg overflow-y-auto rounded-[var(--radius-card)] bg-white p-6">
      <div class="flex items-start justify-between">
        <div>
          <p class="eyebrow mb-1">PRECISA DE AJUDA?</p>
          <h2 class="font-serif text-xl font-bold text-ink">Dúvidas frequentes</h2>
        </div>
        <button class="text-ink-soft hover:text-ink" aria-label="Fechar" @click="open = false"><X :size="20" /></button>
      </div>

      <div class="mt-5 space-y-2">
        <div v-for="(f, i) in faqs" :key="i" class="rounded-xl border border-line/80">
          <button class="flex w-full items-center justify-between gap-3 px-4 py-3 text-left" @click="toggle(i)">
            <span class="flex items-center gap-2.5 text-sm font-semibold text-ink">
              <component :is="f.icon" :size="16" class="shrink-0 text-magenta" />
              {{ f.question }}
            </span>
            <ChevronDown :size="16" :class="['shrink-0 text-ink-soft transition-transform', openQuestion === i ? 'rotate-180' : '']" />
          </button>
          <div v-if="openQuestion === i" class="space-y-1.5 px-4 pb-4 pl-[2.6rem] text-xs leading-relaxed text-ink-soft">
            <p v-for="(line, j) in f.answer" :key="j">{{ line }}</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
