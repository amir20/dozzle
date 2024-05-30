import { Container } from "@/models/Container";

export class Stack {
  constructor(
    public readonly name: string,
    public readonly containers: Container[],
    public readonly services: Service[],
  ) {
    for (const service of services) {
      service.stack = this;
    }
  }

  get updatedAt() {
    return this.containers.map((c) => c.created).reduce((acc, date) => (date > acc ? date : acc), new Date(0));
  }
}

export class Service {
  constructor(
    public readonly name: string,
    public readonly containers: Container[],
  ) {}

  stack?: Stack;

  get updatedAt() {
    return this.containers.map((c) => c.created).reduce((acc, date) => (date > acc ? date : acc), new Date(0));
  }
}
