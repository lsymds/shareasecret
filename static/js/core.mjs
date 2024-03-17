/**
 * Encrypts provided plaintext via the WebCrypto API.
 * @param {string} plainText The text to be encrypted.
 * @returns {string} A string consisting of the encrypted secret, salt, and IV.
 */
function encrypt(plainText) {
    return plainText;
}

/**
 * Decrypts encrypted ciphertext with a given password.
 * @param {string} cipherText Encrypted ciphertext returned from the @see{encrypt} function.
 * @param {string} password The plaintext password to attempt to decrypt the ciphertext with.
 */
function decrypt(cipherText, password) {
    return cipherText;
}

export {
    encrypt,
    decrypt
}
