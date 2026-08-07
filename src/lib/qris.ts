/**
 * QRIS Utility — Konversi QRIS Statis -> Dinamis
 * 
 * Format QRIS mengikuti standar EMVCo QR Code.
 * Setiap field menggunakan TLV (Tag-Length-Value):
 *   [Tag 2 char][Length 2 char][Value]
 * 
 * Tag penting:
 *   00 = Payload Format Indicator
 *   01 = Point of Initiation Method (11=statis, 12=dinamis)
 *   26-51 = Merchant Account Information
 *   52 = Merchant Category Code
 *   53 = Transaction Currency (360=IDR)
 *   54 = Transaction Amount
 *   58 = Country Code (ID)
 *   59 = Merchant Name
 *   60 = Merchant City
 *   63 = CRC (CRC-16/CCITT-FALSE)
 */

// --- TLV Parser ---------------------------------------------------------------

interface TLVField {
  tag: string;
  value: string;
}

function parseTLV(data: string): TLVField[] {
  const fields: TLVField[] = [];
  let i = 0;
  while (i + 4 <= data.length) {
    const tag = data.substring(i, i + 2);
    const length = parseInt(data.substring(i + 2, i + 4), 10);
    if (isNaN(length) || i + 4 + length > data.length) break;
    const value = data.substring(i + 4, i + 4 + length);
    fields.push({ tag, value });
    i += 4 + length;
  }
  return fields;
}

function buildTLV(fields: TLVField[]): string {
  return fields
    .map(f => f.tag + f.value.length.toString().padStart(2, '0') + f.value)
    .join('');
}

// --- CRC-16/CCITT-FALSE ------------------------------------------------------

function crc16ccitt(data: string): string {
  let crc = 0xFFFF;
  for (let i = 0; i < data.length; i++) {
    crc ^= data.charCodeAt(i) << 8;
    for (let j = 0; j < 8; j++) {
      if (crc & 0x8000) {
        crc = (crc << 1) ^ 0x1021;
      } else {
        crc = crc << 1;
      }
      crc &= 0xFFFF;
    }
  }
  return crc.toString(16).toUpperCase().padStart(4, '0');
}

// --- Validate QRIS -----------------------------------------------------------

/**
 * Validate a QRIS string by checking its CRC.
 */
export function validateQris(qrisString: string): boolean {
  if (!qrisString || qrisString.length < 10) return false;
  
  // CRC is the last 4 characters
  const dataWithoutCrc = qrisString.slice(0, -4);
  const existingCrc = qrisString.slice(-4).toUpperCase();
  
  // The CRC is calculated over the data + "6304" (tag 63 + length 04)
  // But since tag 63 is already included in the data, we check differently
  const calculated = crc16ccitt(dataWithoutCrc);
  return calculated === existingCrc;
}

// --- Static -> Dynamic Conversion ---------------------------------------------

/**
 * Convert a static QRIS string to a dynamic one with a specific amount.
 * 
 * Steps:
 * 1. Parse TLV fields
 * 2. Change tag 01 (Point of Initiation Method) from "11" to "12"
 * 3. Add/update tag 54 (Transaction Amount) with the amount
 * 4. Remove old CRC (tag 63)
 * 5. Rebuild the string
 * 6. Calculate new CRC and append
 * 
 * @param staticQris - The base static QRIS string from your bank
 * @param amount - Transaction amount in IDR (integer)
 * @returns Dynamic QRIS string with amount embedded
 */
export function staticToDynamic(staticQris: string, amount: number): string {
  if (!staticQris || amount <= 0) {
    throw new Error('Invalid QRIS string or amount');
  }

  const fields = parseTLV(staticQris);
  if (fields.length === 0) {
    throw new Error('Failed to parse QRIS TLV structure');
  }

  // Remove CRC field (tag 63) - we'll recalculate it
  const fieldsWithoutCrc = fields.filter(f => f.tag !== '63');

  // Update/add fields
  const updatedFields: TLVField[] = [];
  let hasTag01 = false;
  let hasTag54 = false;

  for (const field of fieldsWithoutCrc) {
    if (field.tag === '01') {
      // Change Point of Initiation Method: 11 (static) -> 12 (dynamic)
      updatedFields.push({ tag: '01', value: '12' });
      hasTag01 = true;
    } else if (field.tag === '54') {
      // Update Transaction Amount
      updatedFields.push({ tag: '54', value: amount.toString() });
      hasTag54 = true;
    } else {
      updatedFields.push(field);
    }
  }

  // If tag 01 didn't exist, add it after tag 00
  if (!hasTag01) {
    const idx = updatedFields.findIndex(f => f.tag === '00');
    updatedFields.splice(idx + 1, 0, { tag: '01', value: '12' });
  }

  // If tag 54 didn't exist, add it after tag 53 (currency) or before tag 58 (country)
  if (!hasTag54) {
    const idx53 = updatedFields.findIndex(f => f.tag === '53');
    const idx58 = updatedFields.findIndex(f => f.tag === '58');
    const insertAt = idx53 >= 0 ? idx53 + 1 : (idx58 >= 0 ? idx58 : updatedFields.length);
    updatedFields.splice(insertAt, 0, { tag: '54', value: amount.toString() });
  }

  // Build TLV string without CRC
  const tlvString = buildTLV(updatedFields);

  // Add CRC tag header (63 + 04) then calculate CRC over everything
  const withCrcHeader = tlvString + '6304';
  const crc = crc16ccitt(withCrcHeader);

  return withCrcHeader + crc;
}

// --- Extract Merchant Info ----------------------------------------------------

/**
 * Extract merchant info from a QRIS string for display purposes.
 */
export function extractMerchantInfo(qrisString: string): {
  merchantName: string;
  merchantCity: string;
  merchantId: string;
  isValid: boolean;
} {
  const fields = parseTLV(qrisString);
  
  const tag59 = fields.find(f => f.tag === '59')?.value || '';
  const tag60 = fields.find(f => f.tag === '60')?.value || '';
  
  // Merchant Account Info is usually in tags 26-51
  let merchantId = '';
  for (const f of fields) {
    const tagNum = parseInt(f.tag, 10);
    if (tagNum >= 26 && tagNum <= 51) {
      // Parse sub-TLV to find merchant ID (sub-tag 02 or 03)
      const subFields = parseTLV(f.value);
      const subTag02 = subFields.find(sf => sf.tag === '02')?.value;
      const subTag03 = subFields.find(sf => sf.tag === '03')?.value;
      merchantId = subTag02 || subTag03 || '';
      if (merchantId) break;
    }
  }

  return {
    merchantName: tag59,
    merchantCity: tag60,
    merchantId,
    isValid: validateQris(qrisString),
  };
}
