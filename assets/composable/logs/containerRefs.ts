import type { Container } from "@/models/Container";

/** Names each container as `container=<host>/<id>`, what the server's parseContainerRefs reads. */
export function appendContainerRefs(params: URLSearchParams, containers: Container[]) {
  for (const c of containers) params.append("container", `${c.host}/${c.id}`);
}
