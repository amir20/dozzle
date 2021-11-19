export interface Container {
  id: string;
  created: number;
  image: string;
  name: string;
  state: string;
  status: string;
  stat: ContainerStat;
}

export interface ContainerStat {
  cpu: number;
  memory: number;
  memoryUsage: number;
}
