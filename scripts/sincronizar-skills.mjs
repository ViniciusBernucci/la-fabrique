#!/usr/bin/env node
// Distribui as skills sem alterar nenhum byte da fonte.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const source = path.join(root, 'skills');
const targets = ['.agents/skills', '.claude/skills', '.agent/skills'];
const args = process.argv.slice(2);
if (args.length !== 1 || !['--check', '--apply'].includes(args[0])) {
  console.error('Uso: node scripts/sincronizar-skills.mjs --check|--apply');
  process.exit(2);
}
const apply = args[0] === '--apply';

function filesIn(folder, relative = '') {
  return fs.readdirSync(folder, { withFileTypes: true }).flatMap(entry => {
    const name = path.join(relative, entry.name);
    if (entry.isSymbolicLink()) throw new Error(`Link inesperado: ${folder}/${entry.name}`);
    if (entry.isDirectory()) return filesIn(path.join(folder, entry.name), name);
    if (!entry.isFile()) throw new Error(`Tipo de arquivo inesperado: ${name}`);
    return [name];
  }).sort();
}

// Inclui _shared e todos os recursos para preservar os caminhos relativos.
const files = filesIn(source);
const expected = new Set(files);
const skills = fs.readdirSync(source).filter(name =>
  fs.existsSync(path.join(source, name, 'SKILL.md')));
let differences = 0;
let extras = 0;
for (const target of targets) {
  const destination = path.join(root, target);
  // Recusa links na cadeia do destino para não escrever fora do repositório.
  for (const folder of [path.dirname(destination), destination]) {
    if (fs.existsSync(folder) && fs.lstatSync(folder).isSymbolicLink()) {
      throw new Error(`Destino não pode ser link: ${folder}`);
    }
  }
  const existing = fs.existsSync(destination) ? filesIn(destination) : [];
  for (const name of existing.filter(name => !expected.has(name))) {
    console.error(`Arquivo extra (preservado): ${target}/${name}`);
    extras++;
  }
  for (const name of files) {
    const original = fs.readFileSync(path.join(source, name));
    const output = path.join(destination, name);
    if (fs.existsSync(output) && original.equals(fs.readFileSync(output))) continue;
    differences++;
    if (apply) {
      fs.mkdirSync(path.dirname(output), { recursive: true });
      fs.copyFileSync(path.join(source, name), output);
    } else {
      console.error(`Ausente ou diferente: ${target}/${name}`);
    }
  }
}
console.log(`${skills.length} skills; ${files.length} arquivos por destino; ${targets.length} destinos; ${differences} ${apply ? 'arquivos sincronizados' : 'divergências'}; ${extras} extras.`);
process.exitCode = extras || (!apply && differences) ? 1 : 0;
