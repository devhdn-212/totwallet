import Decimal from 'decimal.js';

export function getInputValue(e: Event): string {
  return (e.target as HTMLInputElement).value;
}

// Convert ke lowercase dan remove non-alphanumeric
export function sanitizeUsername(value: string): string {
  return value.toLowerCase().replace(/[^a-z0-9]/g, "");
}
export function sanitizeidlower(value: string): string {
  return value.toLowerCase().replace(/[^a-z0-9]/g, "");
}
export function sanitizeidupper(value: string): string {
  return value.toUpperCase().replace(/[^A-Z0-9]/g, "");
}
export function sanitizenumber(value: string): string {
  return value.toUpperCase().replace(/[^0-9]/g, "");
}
export function sanitizefloat(value: string): string {
  // 1. Minus valid cuma kalau di posisi paling depan
  const isNegative = value.trimStart().startsWith("-");

  // 2. Buang semua selain angka dan titik
  let v = value.replace(/[^0-9.]/g, "");

  // 3. Titik di depan → kasih 0 (".5" → "0.5"), bukan dibuang
  if (v.startsWith(".")) {
    v = "0" + v;
  }

  // 4. Hanya satu titik
  const firstDot = v.indexOf(".");
  if (firstDot !== -1) {
    v = v.slice(0, firstDot + 1) + v.slice(firstDot + 1).replace(/\./g, "");
  }

  // 5. Pasang minus balik — termasuk saat v masih kosong (user baru ketik "-")
  return isNegative ? "-" + v : v;
}
export function sanitizestringnormal(value: string): string {
  return value.replace(/[^a-zA-Z0-9. ]/g, "");
}
export function sanitizestringtext(value: string): string {
  return value.replace(/[^a-zA-Z0-9:?!.,=> \n]/g, "");
}
export function sanitizestringurl(value: string): string {
  const trimmed = value.trim();

  try {
    const url = new URL(trimmed);

    // hanya boleh http / https
    if (url.protocol !== "http:" && url.protocol !== "https:") {
      return "";
    }

    return url.toString();
  } catch {
    return "";
  }
}
export function decimal(str: string): string {
	if (!str || str === '.' || str === '-') return '0';
	try {
		const num = new Decimal(str);
		return num.toNumber().toLocaleString('id-ID', { maximumFractionDigits: 20 });
	} catch {
		return '0';
	}
}