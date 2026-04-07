import Docker from "dockerode";
import { DEADLOCK_IMAGE } from "./config";

const docker = new Docker({ socketPath: "/var/run/docker.sock" });

export interface ContainerInfo {
  id: string;
  status: string;
  state: string;
  startedAt: string;
}

export interface ContainerStats {
  cpuPercent: number;
  memoryMb: number;
  memoryLimitMb: number;
}

export async function getContainerInfo(containerId: string): Promise<ContainerInfo | null> {
  try {
    const container = docker.getContainer(containerId);
    const info = await container.inspect();
    return {
      id: info.Id.slice(0, 12),
      status: info.State.Status,
      state: info.State.Status,
      startedAt: info.State.StartedAt,
    };
  } catch {
    return null;
  }
}

export async function getContainerStats(containerId: string): Promise<ContainerStats | null> {
  try {
    const container = docker.getContainer(containerId);
    const stats = await container.stats({ stream: false });

    const cpuDelta = stats.cpu_stats.cpu_usage.total_usage - stats.precpu_stats.cpu_usage.total_usage;
    const systemDelta = stats.cpu_stats.system_cpu_usage - stats.precpu_stats.system_cpu_usage;
    const numCpus = stats.cpu_stats.online_cpus || 1;
    const cpuPercent = systemDelta > 0 ? (cpuDelta / systemDelta) * numCpus * 100 : 0;

    const memoryMb = (stats.memory_stats.usage || 0) / 1024 / 1024;
    const memoryLimitMb = (stats.memory_stats.limit || 0) / 1024 / 1024;

    return { cpuPercent: Math.round(cpuPercent * 10) / 10, memoryMb: Math.round(memoryMb), memoryLimitMb: Math.round(memoryLimitMb) };
  } catch {
    return null;
  }
}

export async function createContainer(opts: {
  name: string;
  port: number;
  env: Record<string, string>;
  volumePath: string;
}): Promise<string> {
  const envArray = Object.entries(opts.env).map(([k, v]) => `${k}=${v}`);
  envArray.push("PROTON_LOG=1", "PROTON_NO_WRITE_WATCH=1", "WINEDLLOVERRIDES=winedbg.exe=d", "DISPLAY=:99");

  const container = await docker.createContainer({
    Image: DEADLOCK_IMAGE,
    name: opts.name,
    Env: envArray,
    HostConfig: {
      Binds: [
        `${opts.volumePath}:/app`,
        "/etc/localtime:/etc/localtime:ro",
        "/etc/machine-id:/etc/machine-id:ro",
      ],
      PortBindings: {
        [`${opts.port}/tcp`]: [{ HostPort: String(opts.port) }],
        [`${opts.port}/udp`]: [{ HostPort: String(opts.port) }],
      },
      RestartPolicy: { Name: "always" },
    },
    ExposedPorts: {
      [`${opts.port}/tcp`]: {},
      [`${opts.port}/udp`]: {},
    },
  });

  return container.id;
}

export async function startContainer(containerId: string) {
  await docker.getContainer(containerId).start();
}

export async function stopContainer(containerId: string) {
  await docker.getContainer(containerId).stop();
}

export async function restartContainer(containerId: string) {
  await docker.getContainer(containerId).restart();
}

export async function removeContainer(containerId: string) {
  try {
    await docker.getContainer(containerId).stop();
  } catch { /* already stopped */ }
  await docker.getContainer(containerId).remove();
}

export { docker };
