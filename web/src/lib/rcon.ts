import { Rcon } from "rcon-client";

export async function sendRconCommand(port: number, command: string, password: string = ""): Promise<string> {
  const rcon = await Rcon.connect({
    host: "127.0.0.1",
    port: port,
    password: password,
  });

  try {
    const response = await rcon.send(command);
    return response;
  } finally {
    rcon.end();
  }
}
