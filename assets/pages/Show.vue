<script lang="ts" setup>
import { useContainerStore } from "@/stores/container";
import { storeToRefs } from "pinia";
import { watch } from "vue";
import { useRoute, useRouter } from "vue-router";

const router = useRouter();
const route = useRoute();

const store = useContainerStore();
const { visibleContainers } = storeToRefs(store);

watch(visibleContainers, (newValue) => {
  if (newValue) {
    if (route.query.name) {
      const [container, _] = visibleContainers.value.filter((c) => c.name == route.query.name);
      if (container) {
        router.push({ name: "container", params: { id: container.id } });
      } else {
        console.error(`No containers found matching name=${route.query.name}. Redirecting to /`);
        router.push({ name: "default" });
      }
    } else {
      console.error(`Expection query parameter name to be set. Redirecting to /`);
      router.push({ name: "default" });
    }
  }
});
</script>
