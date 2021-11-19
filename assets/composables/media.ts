import { useMediaQuery } from "@vueuse/core";

export const isMobile = useMediaQuery("(max-width: 770px)");
