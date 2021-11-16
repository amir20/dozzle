import { Ref , computed} from "vue";
import { useStore } from "vuex";

export default function useContainer(id: Ref<string>) {
  const store = useStore();
  const container = computed(() => store.getters.allContainersById[id.value]);

  return { container };
}
