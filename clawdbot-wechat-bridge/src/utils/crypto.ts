/**
 * WeChat Message Encryption/Decryption (AES-256-CBC)
 * 
 * Reference: https://developers.weixin.qq.com/doc/oplatform/Third-party_Platforms/2.0/api/Before_Develop/Technical_Preparation.html
 */

import crypto from 'crypto';
import { XMLParser } from 'fast-xml-parser';

const xmlParser = new XMLParser({
    ignoreAttributes: true,
    parseTagValue: true,
});

/**
 * Decode Base64 encoded EncodingAESKey to get the actual AES key
 * EncodingAESKey is Base64 encoded, 43 characters, actual key is 32 bytes
 */
function decodeAESKey(encodingAESKey: string): Buffer {
    return Buffer.from(encodingAESKey + '=', 'base64');
}

/**
 * PKCS#7 unpadding
 */
function pkcs7Unpad(buf: Buffer): Buffer {
    const padLen = buf[buf.length - 1];
    if (padLen < 1 || padLen > 32) {
        return buf;
    }
    return buf.subarray(0, buf.length - padLen);
}

/**
 * PKCS#7 padding
 */
function pkcs7Pad(buf: Buffer, blockSize: number = 32): Buffer {
    const padLen = blockSize - (buf.length % blockSize);
    const padding = Buffer.alloc(padLen, padLen);
    return Buffer.concat([buf, padding]);
}

/**
 * Decrypt WeChat encrypted message
 * 
 * @param encrypted - Base64 encoded encrypted message
 * @param encodingAESKey - The EncodingAESKey from WeChat settings (43 chars)
 * @param appId - The AppID for verification
 * @returns Decrypted XML message string
 */
export function decryptMessage(
    encrypted: string,
    encodingAESKey: string,
    appId: string
): string {
    const aesKey = decodeAESKey(encodingAESKey);
    const iv = aesKey.subarray(0, 16);

    const encryptedBuffer = Buffer.from(encrypted, 'base64');

    const decipher = crypto.createDecipheriv('aes-256-cbc', aesKey, iv);
    decipher.setAutoPadding(false);

    const decryptedRaw = Buffer.concat([
        decipher.update(encryptedBuffer),
        decipher.final(),
    ]);

    // Remove PKCS#7 padding
    const decrypted = pkcs7Unpad(decryptedRaw);

    // Message format: random(16) + msg_len(4) + msg + appid
    // First 16 bytes: random
    // Next 4 bytes: message length (big-endian)
    const msgLen = decrypted.readUInt32BE(16);
    const msgStart = 20;
    const msgEnd = msgStart + msgLen;

    const message = decrypted.subarray(msgStart, msgEnd).toString('utf-8');
    const extractedAppId = decrypted.subarray(msgEnd).toString('utf-8');

    // Verify AppID
    if (extractedAppId !== appId) {
        throw new Error(`AppID mismatch: expected ${appId}, got ${extractedAppId}`);
    }

    return message;
}

/**
 * Encrypt message for WeChat
 * 
 * @param message - Plain XML message
 * @param encodingAESKey - The EncodingAESKey from WeChat settings
 * @param appId - The AppID
 * @returns Base64 encoded encrypted message
 */
export function encryptMessage(
    message: string,
    encodingAESKey: string,
    appId: string
): string {
    const aesKey = decodeAESKey(encodingAESKey);
    const iv = aesKey.subarray(0, 16);

    // Random 16 bytes
    const random = crypto.randomBytes(16);

    // Message buffer
    const msgBuffer = Buffer.from(message, 'utf-8');

    // Message length (4 bytes, big-endian)
    const msgLen = Buffer.alloc(4);
    msgLen.writeUInt32BE(msgBuffer.length, 0);

    // AppID buffer
    const appIdBuffer = Buffer.from(appId, 'utf-8');

    // Combine: random + msgLen + msg + appId
    const plaintext = Buffer.concat([random, msgLen, msgBuffer, appIdBuffer]);

    // PKCS#7 padding
    const padded = pkcs7Pad(plaintext);

    const cipher = crypto.createCipheriv('aes-256-cbc', aesKey, iv);
    cipher.setAutoPadding(false);

    const encrypted = Buffer.concat([cipher.update(padded), cipher.final()]);

    return encrypted.toString('base64');
}

/**
 * Generate signature for encrypted message
 */
export function generateMsgSignature(
    token: string,
    timestamp: string,
    nonce: string,
    encrypted: string
): string {
    const arr = [token, timestamp, nonce, encrypted].sort();
    const str = arr.join('');
    return crypto.createHash('sha1').update(str).digest('hex');
}

/**
 * Validate msg_signature for encrypted message
 */
export function validateMsgSignature(
    token: string,
    timestamp: string,
    nonce: string,
    encrypted: string,
    msgSignature: string
): boolean {
    const calculatedSignature = generateMsgSignature(token, timestamp, nonce, encrypted);
    return calculatedSignature === msgSignature;
}

/**
 * Parse encrypted XML to extract the Encrypt field
 */
export function extractEncryptedContent(xml: string): string {
    const parsed = xmlParser.parse(xml);
    return parsed.xml?.Encrypt || '';
}

/**
 * Build encrypted response XML
 */
export function buildEncryptedReply(
    encrypted: string,
    signature: string,
    timestamp: string,
    nonce: string
): string {
    return `<xml>
<Encrypt><![CDATA[${encrypted}]]></Encrypt>
<MsgSignature><![CDATA[${signature}]]></MsgSignature>
<TimeStamp>${timestamp}</TimeStamp>
<Nonce><![CDATA[${nonce}]]></Nonce>
</xml>`;
}
