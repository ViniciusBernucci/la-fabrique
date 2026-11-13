import { createHmac } from "node:crypto";

const ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";

function base32Decode(input: string): Buffer {
  let bits = 0;
  let value = 0;
  const out: number[] = [];
  for (const ch of input.replace(/=+$/g, "").replace(/\s+/g, "").toUpperCase()) {
    const idx = ALPHABET.indexOf(ch);
    if (idx < 0) throw new Error("chave TOTP inválida");
    value = (value << 5) | idx;
    bits += 5;
    if (bits >= 8) {
      out.push((value >>> (bits - 8)) & 0xff);
      bits -= 8;
    }
  }
  return Buffer.from(out);
}

/** Código TOTP (RFC 6238: SHA-1, 6 dígitos, janela de 30 s). */
export function totp(secret: string, at: Date = new Date()): string {
  const counter = Math.floor(at.getTime() / 1000 / 30);
  const buf = Buffer.alloc(8);
  buf.writeBigUInt64BE(BigInt(counter));
  const h = createHmac("sha1", base32Decode(secret)).update(buf).digest();
  const offset = (h[h.length - 1] ?? 0) & 0x0f;
  const bin =
    (((h[offset] ?? 0) & 0x7f) << 24) |
    (((h[offset + 1] ?? 0) & 0xff) << 16) |
    (((h[offset + 2] ?? 0) & 0xff) << 8) |
    ((h[offset + 3] ?? 0) & 0xff);
  return String(bin % 1_000_000).padStart(6, "0");
}
