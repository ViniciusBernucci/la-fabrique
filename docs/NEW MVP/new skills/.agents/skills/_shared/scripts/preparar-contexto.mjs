#!/usr/bin/env node
// Ponto único após criar/reutilizar worktree: não cria branches nem altera Git rastreado.
import fs from 'node:fs';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
const cli = process.env.CONTEXTO_CLI || fileURLToPath(new URL('../../../contexto/contexto.mjs', import.meta.url));
if (!fs.existsSync(cli)) throw Error('Contexto não instalado ao lado das skills; configure CONTEXTO_CLI com o caminho real');
const result = spawnSync(process.execPath, [cli, 'ensure', ...process.argv.slice(2)], { stdio: 'inherit', windowsHide: true });
if (result.error) throw result.error;
process.exitCode = result.status ?? 1;
