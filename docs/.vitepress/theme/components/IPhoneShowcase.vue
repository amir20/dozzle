<script lang="ts" setup>
import { computed } from "vue";
import { useRoute, withBase } from "vitepress";

// Dozzle Cloud for iPhone, shown on the home page: the words and the way into
// the beta on one side, three real screens fanned out on the other. The panel
// stays dark in both themes, like the app. Copy lives here, like
// HeroAnnouncement's, so the five home pages do not repeat it.
const IOS_URL = "https://cloud.dozzle.dev/ios?from=dozzle_docs_showcase";
const COPY = {
  en: {
    eyebrow: "Dozzle Cloud for iPhone",
    badge: "New · Private beta",
    title: "Your containers, in your pocket.",
    body: "Alerts as push notifications, with Restart and Mute right on them. Ask about a container from anywhere.",
    cta: "Request early access",
  },
  de: {
    eyebrow: "Dozzle Cloud fürs iPhone",
    badge: "Neu · Private Beta",
    title: "Deine Container, immer in der Tasche.",
    body: "Warnungen als Push-Benachrichtigungen, mit Neustart und Stummschalten direkt daran. Frag von überall nach einem Container.",
    cta: "Frühen Zugang anfragen",
  },
  fr: {
    eyebrow: "Dozzle Cloud pour iPhone",
    badge: "Nouveau · Bêta privée",
    title: "Vos conteneurs, dans votre poche.",
    body: "Les alertes arrivent en notifications push, avec Redémarrer et Couper le son directement dessus. Posez une question sur un conteneur, où que vous soyez.",
    cta: "Demander un accès anticipé",
  },
  es: {
    eyebrow: "Dozzle Cloud para iPhone",
    badge: "Nuevo · Beta privada",
    title: "Tus contenedores, en el bolsillo.",
    body: "Las alertas llegan como notificaciones push, con Reiniciar y Silenciar ahí mismo. Pregunta por un contenedor desde donde estés.",
    cta: "Solicitar acceso anticipado",
  },
  zh: {
    eyebrow: "Dozzle Cloud iPhone 版",
    badge: "新功能 · 私密测试",
    title: "你的容器，尽在掌中。",
    body: "告警直接以推送通知送达，通知上就能重启或静音。不管在哪，都能随时询问某个容器的情况。",
    cta: "申请抢先体验",
  },
} as const;

type Locale = keyof typeof COPY;

const route = useRoute();
const copy = computed(() => {
  const segment = route.path.split("/")[1] as Locale;
  return COPY[segment] ?? COPY.en;
});

// Front phone last so it paints on top. The two behind it lean out to either side.
const phones = [
  { key: "alert", class: "w-[34%] opacity-80", style: "transform: translateX(-118%) translateY(10%) rotate(-8deg)" },
  { key: "ask", class: "w-[34%] opacity-80", style: "transform: translateX(18%) translateY(10%) rotate(8deg)" },
  { key: "overview", class: "w-[40%]", style: "transform: translateX(-50%)" },
];
</script>

<template>
  <section class="mx-auto mt-20 max-w-6xl px-6">
    <div
      class="relative grid overflow-hidden rounded-2xl bg-neutral-950 ring-1 ring-white/10 md:grid-cols-[1fr_1.15fr]"
    >
      <div class="absolute -top-32 -left-24 size-96 rounded-full bg-(--vp-c-brand-1)/35 blur-3xl" />
      <div class="absolute -right-24 -bottom-24 size-96 rounded-full bg-sky-600/30 blur-3xl" />

      <div class="relative px-8 pt-12 text-center md:py-16 md:pl-14 md:text-left">
        <div class="mb-6 flex items-center justify-center gap-3 md:justify-start">
          <img :src="withBase('/ios/icon.png')" alt="" class="size-12 rounded-[22%] shadow-lg ring-1 ring-white/10" />
          <div class="text-left">
            <p class="m-0! text-sm font-semibold text-white">{{ copy.eyebrow }}</p>
            <p class="m-0! text-xs font-medium text-indigo-300">{{ copy.badge }}</p>
          </div>
        </div>
        <h2 class="leading-1.1! m-0! border-none! p-0! text-4xl! font-semibold tracking-tight text-white! md:text-5xl!">
          {{ copy.title }}
        </h2>
        <p class="mx-auto mt-4! mb-8! max-w-md text-lg leading-relaxed text-white/70 md:mx-0">
          {{ copy.body }}
        </p>
        <a
          :href="IOS_URL"
          class="inline-flex items-center gap-2 rounded-full bg-white px-6 py-2.5 text-sm font-semibold text-neutral-950! no-underline! transition-colors hover:bg-white/85"
        >
          {{ copy.cta }}
          <Icon icon="mdi:arrow-right" class="size-4" />
        </a>
      </div>

      <div class="relative mt-10 h-80 sm:h-96 md:mt-0 md:h-auto md:min-h-112" aria-hidden="true">
        <div
          v-for="p in phones"
          :key="p.key"
          class="absolute top-6 left-1/2 max-w-60 rounded-[2.4rem] bg-neutral-900 p-1.5 shadow-2xl ring-1 ring-white/15 md:top-14"
          :class="p.class"
          :style="p.style"
        >
          <img :src="withBase(`/ios/${p.key}.jpg`)" alt="" loading="lazy" class="block w-full rounded-[2.1rem]" />
        </div>
        <div class="absolute inset-x-0 bottom-0 h-20 bg-linear-to-t from-neutral-950 to-transparent" />
      </div>
    </div>
  </section>
</template>
