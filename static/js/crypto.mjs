/**
 * Encrypts provided plaintext via the WebCrypto API.
 * @param {string} plainText The text to be encrypted.
 * @param {string} password The password to use to encrypt the text.
 * @returns {Promise<string>} A string consisting of the encrypted secret, salt, and IV.
 */
export async function encrypt(plainText, password) {
    const enc = new TextEncoder();
    const salt = window.crypto.getRandomValues(new Uint8Array(16));
    const iv = window.crypto.getRandomValues(new Uint8Array(12));
    const encryptionKey = await _keyFromPassword(password, salt, "encrypt");

    const cipherText = await window.crypto.subtle.encrypt(
        { name: "AES-GCM", iv },
        encryptionKey,
        enc.encode(plainText)
    );

    return `${_arrayBufferToBase64String(new Uint8Array(cipherText))}.${_arrayBufferToBase64String(salt)}.${_arrayBufferToBase64String(iv)}`;
}

/**
 * Decrypts encrypted ciphertext with a given password.
 * @param {string} cipherText Encrypted ciphertext returned from the encrypt function.
 * @param {string} password The plaintext password to attempt to decrypt the ciphertext with.
 * @returns {Promise<string>} The decrypted text.
 */
export async function decrypt(cipherText, password) {
    if (!cipherText) {
        return;
    }

    const encryptionComponents = cipherText.split(".");
    if (encryptionComponents.length !== 3) {
        return;
    }

    const [encryptedContentText, saltText, ivText] = encryptionComponents;
    const encryptedContent = _base64StringToArrayBuffer(encryptedContentText);
    const salt = _base64StringToArrayBuffer(saltText);
    const iv = _base64StringToArrayBuffer(ivText);

    const decryptionKey = await _keyFromPassword(password, salt, "decrypt");

    const decryptedBuffer = await window.crypto.subtle.decrypt(
        { name: "AES-GCM", iv },
        decryptionKey,
        encryptedContent
    );

    return new TextDecoder().decode(decryptedBuffer);
}

/**
 * Dervies a cryptographically secure encryption key from a password using the PBKDF hashing algorithm.
 * @param {string} password The password to derive the key from.
 * @param {Uint8Array} salt A cryptographically-secure randomly generated salt.
 * @param {string} use What the derived key will be used for.
 * @returns {Promise<CryptoKey>} The created key.
 */
async function _keyFromPassword(password, salt, use = "encrypt") {
    const enc = new TextEncoder();

    const material = await window.crypto.subtle.importKey(
      "raw",
      enc.encode(password),
      "PBKDF2",
      false,
      ["deriveKey"]
    );

    const derivedKey = await window.crypto.subtle.deriveKey(
      {
        name: "PBKDF2",
        salt,
        iterations: 600000,
        hash: "SHA-256"
      },
      material,
      { name: "AES-GCM", length: 256 },
      false,
      [use]
    );

    return derivedKey;
}

/**
 * Converts an ArrayBuffer to a base64 encoded string.
 * @param {Uint8Array} buffer The buffer to convert each value to a string.
 * @returns {string} A base64 encoded string representation of the buffer.
 */
function _arrayBufferToBase64String(buffer) {
    const binary = Array.prototype.map
        .call(buffer, (byte) => String.fromCharCode(byte))
        .join("");

    return btoa(binary);
}

/**
 * Converts a Base64 encoded string to an ArrayBuffer.
 * @param {string} str The base64 encoded string to convert to an array buffer.
 * @returns {Uint8Array}
 */
function _base64StringToArrayBuffer(str) {
    return Uint8Array.from(atob(str), c => c.charCodeAt(0))
}
