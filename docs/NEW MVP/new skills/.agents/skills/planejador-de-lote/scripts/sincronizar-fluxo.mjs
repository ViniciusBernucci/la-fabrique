#!/usr/bin/env node
// Sincronização portátil. Sem Git de escrita; --check é o padrão.
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import crypto from 'node:crypto';
import { fileURLToPath } from 'node:url';

const central = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
const project = path.dirname(central), dev = path.dirname(project);
const apply = process.argv.includes('--apply'), global = process.argv.includes('--global');
const skills = fs.readdirSync(path.join(central, 'skills')).filter(name => fs.existsSync(path.join(central, 'skills', name, 'SKILL.md')));
const linkNames = [...skills, '_shared'];
const changes = [];
function exists(file) { try { fs.lstatSync(file); return true; } catch { return false; } }
function same(link, target) {
  try { return fs.realpathSync(link).toLowerCase() === fs.realpathSync(target).toLowerCase(); } catch { return false; }
}
function link(folder, name, target) {
  const destination = path.resolve(folder, name);
  if (path.dirname(destination) !== path.resolve(folder) || !linkNames.includes(name)) throw Error('Destino de sincronização inválido');
  if (same(destination, target)) return;
  changes.push({ path: destination, target });
  if (!apply) return;
  if (exists(destination)) {
    // Move apenas o diretório/link nominal conferido, nunca enumera conteúdo para exclusão.
    const backup = path.join(central, 'backups', `sync-${Date.now()}-${crypto.randomUUID()}`, name);
    if (!backup.startsWith(path.join(central, 'backups') + path.sep)) throw Error('Backup fora da fonte central');
    fs.mkdirSync(path.dirname(backup), { recursive: true });
    fs.renameSync(destination, backup);
  }
  fs.mkdirSync(folder, { recursive: true });
  fs.symlinkSync(target, destination, process.platform === 'win32' ? 'junction' : 'dir');
}
// Corrige cópias locais que sombreiam a fonte. Não toca outras skills como archify.
const local = path.join(project, '.agents', 'skills', 'skills');
for (const name of linkNames) {
  if (fs.existsSync(local)) link(local, name, path.join(central, 'skills', name));
  if (global) for (const homeFolder of ['.agents', '.claude']) link(path.join(os.homedir(), homeFolder, 'skills'), name, path.join(central, 'skills', name));
}
const mirror = path.join(dev, 'intranetNova', 'CLAUDE.md');
if (fs.existsSync(path.dirname(mirror))) {
  const body = fs.readFileSync(path.join(central, 'CLAUDE.md'));
  const previous = fs.existsSync(mirror) ? fs.readFileSync(mirror) : Buffer.alloc(0);
  if (!body.equals(previous)) { changes.push({ path: mirror, mirror: true }); if (apply) fs.writeFileSync(mirror, body); }
}
console.log(JSON.stringify({ mode: apply ? 'apply' : 'check', central, skills: skills.length, changes }, null, 2));
process.exitCode = !apply && changes.length ? 1 : 0;
