import { Container } from "@/models/Container";

export class Stack {
  constructor(
    public readonly name: string,
    public readonly containers: Container[],
    public readonly services: Service[],
  ) {}
}

export class Service {
  constructor(
    public readonly name: string,
    public readonly containers: Container[],
  ) {}
}
